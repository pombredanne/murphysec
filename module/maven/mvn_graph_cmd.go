package maven

import (
	"context"
	"fmt"
	"github.com/murphysecurity/murphysec/env"
	"github.com/murphysecurity/murphysec/infra/logctx"
	"github.com/murphysecurity/murphysec/infra/logpipe"
	"github.com/murphysecurity/murphysec/utils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"strings"
	"time"
)

// PluginGraphCmd helper to com.github.ferstl:depgraph-maven-plugin:4.0.1:graph
type PluginGraphCmd struct {
	Profiles      []string
	Timeout       time.Duration
	ScanDir       string
	MavenCmdInfo  *MvnCommandInfo
	ErrText       string
	TimeoutKilled bool
}

//func multiWriteCloser(writer ...io.Writer)io.WriteCloser {
//
//}

func (m *PluginGraphCmd) RunC(ctx context.Context) error {
	logger := logctx.Use(ctx)
	if ctx == nil {
		ctx = context.TODO()
	}
	if m.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.Timeout)
		defer cancel()
	}
	var args = []string{"com.github.ferstl:depgraph-maven-plugin:4.0.1:graph", "-DgraphFormat=json"}
	if env.TlsAllowInsecure() {
		// https://stackoverflow.com/questions/21252800/how-to-tell-maven-to-disregard-ssl-errors-and-trusting-all-certs
		args = append(args, "-Dmaven.wagon.http.ssl.ignore.validity.dates=true", "-Dmaven.resolver.transport=wagon", "-Dmaven.wagon.http.ssl.allowall=true", "-Dmaven.wagon.http.ssl.insecure=true")
	}

	if len(m.Profiles) > 0 {
		args = append(args, "-P")
		args = append(args, strings.Join(m.Profiles, ","))
	}
	c := m.MavenCmdInfo.Command(ctx, args...)
	c.Dir = m.ScanDir
	utils.SetPGid(c)
	logStream := logpipe.NewWithOption(logpipe.Option{
		Logger: logger,
		Prefix: "mvn",
	})
	lRecorder := launchERecorder()
	defer func() { m.ErrText = lRecorder.String() }()
	var mwriter = utils.MultiWriteCloser(logStream, lRecorder)
	defer func() { _ = mwriter.Close() }()
	c.Stderr = mwriter
	c.Stdout = mwriter
	logger.Info(fmt.Sprintf("Start command: %s", c.String()), zap.String("dir", c.Dir))
	if e := c.Start(); e != nil {
		logger.Error("Start command failed", zap.Error(e))
		return errors.WithMessage(ErrMvnCmd, e.Error())
	}

	timeoutToKilledCh := make(chan bool, 1)
	defer func() { m.TimeoutKilled = <-timeoutToKilledCh }()
	timerCtx, timerCancel := context.WithCancel(ctx)
	defer func() { timerCancel() }()
	go func() {
		var timeoutKilled = false
		defer func() { timeoutToKilledCh <- timeoutKilled }()
		for timerCtx.Err() == nil {
			if logStream.LastLineTimestamp.Load() != nil && time.Since(*logStream.LastLineTimestamp.Load()) > env.CommandTimeout {
				logger.Warn("Maven stop print logs, killed")
				timeoutKilled = true
				utils.KillProcessGroup(c.Process.Pid)
				_ = c.Process.Kill()
			}
			time.Sleep(time.Second)
		}
	}()
	if e := c.Wait(); e != nil {
		logger.Error(ErrMvnCmd.Error(), zap.Error(e), zap.Int("exit_code", c.ProcessState.ExitCode()))
		return errors.WithMessage(ErrMvnCmd, e.Error())
	}
	exitCode := c.ProcessState.ExitCode()
	if exitCode != 0 {
		logger.Error(ErrMvnExitErr.Error(), zap.Int("code", exitCode))
		return errors.WithMessage(ErrMvnExitErr, fmt.Sprintf("code: %d", exitCode))
	}
	logger.Info("Mvn graph command exit with no errors")
	return nil
}
