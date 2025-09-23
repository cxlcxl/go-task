package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"task-executor/define"
	"time"

	"task-executor/config"
	"task-executor/logger"
)

type Executor struct {
	config      *config.Config
	handlers    map[string]JobHandler
	runningJobs map[int]context.CancelFunc
	mutex       sync.RWMutex
}

func NewExecutor(cfg *config.Config) *Executor {
	return &Executor{
		config:      cfg,
		handlers:    make(map[string]JobHandler),
		runningJobs: make(map[int]context.CancelFunc),
	}
}

func RegisterXXLJobHandlers(exec *Executor) {
	// 注册各种任务处理器
	for queueName, handler := range define.XXLJobHandlers {
		exec.RegisterJobHandler(queueName, handler)
	}

	logger.Info("XXL-JOB任务处理器注册成功")
}

// RegisterJobHandler registers a job handler
func (e *Executor) RegisterJobHandler(name string, handler JobHandler) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.handlers[name] = handler
}

// Start starts the executor
func (e *Executor) Start() error {
	// Start XXL-JOB registration in background (non-blocking)
	go e.startRegistryWithRetryLimit()

	// Start heartbeat in background (non-blocking)
	go e.startHeartbeatWithRetryLimit()

	logger.Info("Executor started successfully")
	return nil
}

// Registry with XXL-JOB admin with retry limit
func (e *Executor) startRegistryWithRetryLimit() {
	if e.registryWithResult() {
		logger.Info("成功注册到XXL-JOB管理台")
		// 注册成功后开始定期注册
		e.startPeriodicRegistry()
		return
	}
}

// Registry with result checking
func (e *Executor) registryWithResult() bool {
	registryParam := RegistryParam{
		RegistryGroup: "EXECUTOR",
		RegistryKey:   e.config.XXLJob.AppName,
		RegistryValue: fmt.Sprintf("http://%s:%d", e.config.Executor.IP, e.config.Executor.Port),
	}

	successCount := 0
	for _, adminAddr := range e.config.XXLJob.AdminAddresses {
		url := fmt.Sprintf("%s/api/registry", adminAddr)
		if err := e.postToAdmin(url, registryParam); err != nil {
			logger.Error("Failed to registry to admin %s: %v", adminAddr, err)
		} else {
			logger.Debug("Successfully registered to admin: %s", adminAddr)
			successCount++
		}
	}

	// Return true if at least one admin registration succeeded
	return successCount > 0
}

// Start periodic registry after successful initial registration
func (e *Executor) startPeriodicRegistry() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.registryWithResult()
		}
	}
}

// Heartbeat to XXL-JOB admin with retry limit
func (e *Executor) startHeartbeatWithRetryLimit() {
	// Wait for initial registration to complete
	time.Sleep(8 * time.Second)

	if e.heartbeatWithResult() {
		logger.Debug("心跳成功，开始定期心跳")
		// 心跳成功后开始定期心跳
		e.startPeriodicHeartbeat()
		return
	}
}

// Heartbeat with result checking
func (e *Executor) heartbeatWithResult() bool {
	registryParam := RegistryParam{
		RegistryGroup: "EXECUTOR",
		RegistryKey:   e.config.XXLJob.AppName,
		RegistryValue: fmt.Sprintf("http://%s:%d", e.config.Executor.IP, e.config.Executor.Port),
	}

	successCount := 0
	for _, adminAddr := range e.config.XXLJob.AdminAddresses {
		url := fmt.Sprintf("%s/api/registryRemove", adminAddr)
		if err := e.postToAdmin(url, registryParam); err != nil {
			logger.Debug("Heartbeat failed to admin %s: %v", adminAddr, err)
		} else {
			successCount++
		}
	}

	// Return true if at least one admin heartbeat succeeded
	return successCount > 0
}

// Start periodic heartbeat after successful initial heartbeat
func (e *Executor) startPeriodicHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.heartbeatWithResult()
		}
	}
}

// ExecuteJob Execute job
func (e *Executor) ExecuteJob(param *TriggerParam) *ReturnT {
	e.mutex.RLock()
	handler, exists := e.handlers[param.ExecutorHandler]
	e.mutex.RUnlock()

	if !exists {
		return &ReturnT{
			Code: 500,
			Msg:  fmt.Sprintf("Job handler not found: %s", param.ExecutorHandler),
		}
	}

	// Create job context
	jobCtx := &JobContext{
		JobID:          param.JobID,
		JobParam:       param.ExecutorParams,
		LogID:          param.LogID,
		LogDateTime:    time.Unix(param.LogDateTime/1000, 0),
		BroadcastIndex: param.BroadcastIndex,
		BroadcastTotal: param.BroadcastTotal,
	}

	// Execute job in goroutine
	go func() {
		jobLogger := logger.JobLogger(param.LogID, jobCtx.LogDateTime)
		if jobLogger == nil {
			logger.Error("Failed to create job logger for job %d", param.JobID)
			return
		}

		jobLogger.JobInfo("Job started: %s, params: %s", param.ExecutorHandler, param.ExecutorParams)

		if err := handler.Execute(jobCtx); err != nil {
			jobLogger.JobError("Job failed: %v", err)
			logger.Error("Job %d execution failed: %v", param.JobID, err)
		} else {
			jobLogger.JobInfo("Job completed successfully")
			logger.Info("Job %d completed successfully", param.JobID)
		}
	}()

	return &ReturnT{
		Code: 200,
		Msg:  "success",
	}
}

// IdleBeat checks if executor is idle
func (e *Executor) IdleBeat(param *IdleBeatParam) *ReturnT {
	e.mutex.RLock()
	_, running := e.runningJobs[param.JobID]
	e.mutex.RUnlock()

	if running {
		return &ReturnT{
			Code: 500,
			Msg:  "job is running",
		}
	}

	return &ReturnT{
		Code: 200,
		Msg:  "success",
	}
}

// Kill job
func (e *Executor) Kill(param *KillParam) *ReturnT {
	e.mutex.Lock()
	cancelFunc, exists := e.runningJobs[param.JobID]
	if exists {
		cancelFunc()
		delete(e.runningJobs, param.JobID)
	}
	e.mutex.Unlock()

	return &ReturnT{
		Code: 200,
		Msg:  "success",
	}
}

// Log query
func (e *Executor) Log(param *LogParam) *ReturnT {
	// TODO: Implement log reading logic
	result := &LogResult{
		FromLineNum: param.FromLineNum,
		ToLineNum:   param.FromLineNum,
		LogContent:  "",
		IsEnd:       true,
	}

	return &ReturnT{
		Code:    200,
		Msg:     "success",
		Content: result,
	}
}

func (e *Executor) postToAdmin(url string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if e.config.XXLJob.AccessToken != "" {
		req.Header.Set("XXL-JOB-ACCESS-TOKEN", e.config.XXLJob.AccessToken)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("admin returned status: %d", resp.StatusCode)
	}

	return nil
}
