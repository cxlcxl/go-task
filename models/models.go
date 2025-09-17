package models

import "time"

// TriggerParam represents the trigger parameters from XXL-JOB
type TriggerParam struct {
	JobID                 int    `json:"jobId"`
	ExecutorHandler       string `json:"executorHandler"`
	ExecutorParams        string `json:"executorParams"`
	ExecutorBlockStrategy string `json:"executorBlockStrategy"`
	ExecutorTimeout       int    `json:"executorTimeout"`
	LogID                 int    `json:"logId"`
	LogDateTime           int64  `json:"logDateTime"`
	GlueType              string `json:"glueType"`
	GlueSource            string `json:"glueSource"`
	GlueUpdatetime        int64  `json:"glueUpdatetime"`
	BroadcastIndex        int    `json:"broadcastIndex"`
	BroadcastTotal        int    `json:"broadcastTotal"`
}

// ReturnT represents the return structure for XXL-JOB responses
type ReturnT struct {
	Code    int         `json:"code"`
	Msg     string      `json:"msg"`
	Content interface{} `json:"content"`
}

// RegistryParam represents registry parameters
type RegistryParam struct {
	RegistryGroup string `json:"registryGroup"`
	RegistryKey   string `json:"registryKey"`
	RegistryValue string `json:"registryValue"`
}

// IdleBeatParam represents idle beat parameters
type IdleBeatParam struct {
	JobID int `json:"jobId"`
}

// KillParam represents kill job parameters
type KillParam struct {
	JobID int `json:"jobId"`
}

// LogParam represents log query parameters
type LogParam struct {
	LogDateTim  int64 `json:"logDateTim"`
	LogID       int   `json:"logId"`
	FromLineNum int   `json:"fromLineNum"`
}

// LogResult represents log query result
type LogResult struct {
	FromLineNum int    `json:"fromLineNum"`
	ToLineNum   int    `json:"toLineNum"`
	LogContent  string `json:"logContent"`
	IsEnd       bool   `json:"isEnd"`
}

// JobContext represents job execution context
type JobContext struct {
	JobID          int
	JobParam       string
	LogID          int
	LogDateTime    time.Time
	ShardIndex     int
	ShardTotal     int
	BroadcastIndex int
	BroadcastTotal int
}

// JobHandler defines the interface for job handlers
type JobHandler interface {
	Execute(ctx *JobContext) error
}
