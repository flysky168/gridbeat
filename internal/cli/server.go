package cli

//s := api.New(gdb, cfg)
//app := s.App(fiber.New())

// Start audit retention job.
// 启动审计保留周期清理任务。
//stop := make(chan struct{})
//.StartAuditRetentionJob(stop)

// Graceful shutdown on SIGINT/SIGTERM.
// 监听 SIGINT/SIGTERM 实现优雅退出。
//ctx, cancel := context.WithCancel(context.Background())
//defer cancel()

//go func() {
//ch := make(chan os.Signal, 2)
//	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
//	<-ch
//	close(stop)
//	_ = app.ShutdownWithContext(ctx)
//}()
