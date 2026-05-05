package commands

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/appctx"
)

type Task struct {
	ID       int         `json:"id"`
	Status   string      `json:"status"`
	Progress int         `json:"progress"`
	Result   *TaskResult `json:"result"`
	Error    string      `json:"error"`
}

type TaskResult struct {
	MediaIDs []int `json:"media_ids"`
}

type TaskResponse struct {
	Success bool `json:"success"`
	Task    Task `json:"task"`
}

func NewTasksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Inspect async media processing tasks",
	}
	cmd.AddCommand(newTasksShowCmd())
	cmd.AddCommand(newTasksWaitCmd())
	return cmd
}

func newTasksShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show the current state of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			var resp TaskResponse
			if err := a.Client.Get("/tasks/"+args[0], nil, &resp); err != nil {
				return err
			}
			printJSON(resp)
			return nil
		},
	}
}

func newTasksWaitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wait <id>",
		Short: "Poll a task until it completes or fails, then print the result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := app(cmd)
			taskID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("task id must be an integer")
			}
			mediaIDs, err := waitForTask(a, taskID)
			if err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(map[string]any{"media_ids": mediaIDs})
			} else {
				fmt.Printf("Task completed. Media IDs: %v\n", mediaIDs)
			}
			return nil
		},
	}
}

// waitForTask polls GET /tasks/:id until status is "completed" or "failed".
// On success, returns the media IDs from the task result.
func waitForTask(a *appctx.App, taskID int) ([]int, error) {
	idStr := strconv.Itoa(taskID)
	for {
		var resp TaskResponse
		if err := a.Client.Get("/tasks/"+idStr, nil, &resp); err != nil {
			return nil, err
		}
		switch resp.Task.Status {
		case "completed":
			if !a.JSONOutput {
				fmt.Fprintf(os.Stderr, "\033[2K\r")
			}
			if resp.Task.Result == nil {
				return nil, fmt.Errorf("task %d completed but returned no result", taskID)
			}
			return resp.Task.Result.MediaIDs, nil
		case "failed":
			if !a.JSONOutput {
				fmt.Fprintf(os.Stderr, "\033[2K\r")
			}
			return nil, fmt.Errorf("task %d failed: %s", taskID, resp.Task.Error)
		}
		if !a.JSONOutput && resp.Task.Progress > 0 {
			fmt.Fprintf(os.Stderr, "\033[2K\rProcessing... %d%%", resp.Task.Progress)
		}
		time.Sleep(time.Second)
	}
}
