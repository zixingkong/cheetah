package interfacecase

import (
	"github.com/gin-gonic/gin"
	"github.com/test-instructor/cheetah/server/api/v1"
	"github.com/test-instructor/cheetah/server/middleware"
)

type TimerTaskRouter struct {
}

// InitTimerTaskRouter 初始化 TimerTask 路由信息
func (s *TimerTaskRouter) InitTimerTaskRouter(Router *gin.RouterGroup) {
	taskRouter := Router.Group("").Use(middleware.OperationRecord())
	taskRouterWithoutRecord := Router.Group("")
	var taskApi = v1.ApiGroupApp.InterfaceCaseApiGroup.TimerTaskApi
	{
		taskRouter.POST("createTimerTask", taskApi.CreateTimerTask)             // 新建TimerTask
		taskRouter.DELETE("deleteTimerTask", taskApi.DeleteTimerTask)           // 删除TimerTask
		taskRouter.DELETE("deleteTimerTaskByIds", taskApi.DeleteTimerTaskByIds) // 批量删除TimerTask
		taskRouter.PUT("updateTimerTask", taskApi.UpdateTimerTask)              // 更新TimerTask
		taskRouter.GET("findTaskTestCase", taskApi.FindTaskTestCase)
		taskRouter.POST("addTaskTestCase", taskApi.AddTaskTestCase)
		taskRouter.POST("setTaskCase", taskApi.SetTaskCase)
	}
	{
		taskRouterWithoutRecord.GET("findTimerTask", taskApi.FindTimerTask)       // 根据ID获取TimerTask
		taskRouterWithoutRecord.GET("getTimerTaskList", taskApi.GetTimerTaskList) // 获取TimerTask列表
	}
}