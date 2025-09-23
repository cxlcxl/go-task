package gdt

import (
	"task-executor/define"
	"task-executor/logger"
	"time"
)

// EmailJobHandler handles email sending tasks
type EmailJobHandler struct{}

func NewEmailJobHandler() *EmailJobHandler {
	return &EmailJobHandler{}
}

func (h *EmailJobHandler) Execute(ctx *define.JobContext) error {
	logger.Info("Email job started, JobID: %d, Params: %s", ctx.JobID, ctx.JobParam)

	// Simulate email sending
	logger.Info("Sending email...")
	time.Sleep(2 * time.Second)

	// In a real implementation, you would:
	// 1. Parse email parameters from ctx.JobParam
	// 2. Connect to email service
	// 3. Send email
	// 4. Handle errors appropriately

	logger.Info("Email sent successfully")
	return nil
}
