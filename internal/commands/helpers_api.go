package commands

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func confirmAction(question string, force bool) bool {
	if force {
		return true
	}
	fmt.Printf("%s Pass --force to skip this prompt.\n", question)
	var answer string
	fmt.Print("Type 'yes' to confirm: ")
	fmt.Scanln(&answer)
	if answer != "yes" {
		fmt.Println("Aborted.")
		return false
	}
	return true
}

// runDelete sends DELETE and reports the API's {success, message} answer.
func runDelete(cmd *cobra.Command, path, label string) error {
	a := app(cmd)
	var resp map[string]any
	if err := a.Client.DeleteWithBody(path, nil, &resp); err != nil {
		return err
	}
	if a.JSONOutput {
		printJSON(resp)
	}
	message, _ := resp["message"].(string)
	if success, ok := resp["success"].(bool); ok && !success {
		if message == "" {
			message = "delete failed"
		}
		return fmt.Errorf("%s: %s", label, message)
	}
	if a.JSONOutput {
		return nil
	}
	if message != "" {
		fmt.Printf("%s: %s\n", label, message)
	} else {
		fmt.Printf("Deleted %s\n", label)
	}
	return nil
}

// sendJSON sends a request and prints the answer as JSON.
func sendJSON(cmd *cobra.Command, method, path string, body any) error {
	a := app(cmd)
	var resp map[string]any
	var err error
	switch method {
	case "GET":
		err = a.Client.Get(path, nil, &resp)
	case "POST":
		err = a.Client.Post(path, body, &resp)
	case "PATCH":
		err = a.Client.Patch(path, body, &resp)
	case "PUT":
		err = a.Client.Put(path, body, &resp)
	case "DELETE":
		err = a.Client.DeleteWithBody(path, body, &resp)
	default:
		err = fmt.Errorf("unsupported method %s", method)
	}
	if err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

func getJSON(cmd *cobra.Command, path string, q url.Values) error {
	a := app(cmd)
	var resp any
	if err := a.Client.Get(path, q, &resp); err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

func addFieldsFlag(cmd *cobra.Command, fields *string) {
	cmd.Flags().StringVar(fields, "fields", "", "Comma-separated fields to include")
}

func setIfChanged(cmd *cobra.Command, body map[string]any, flag, key string, value any) {
	if cmd.Flags().Changed(flag) {
		body[key] = value
	}
}

func setStringIf(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func atoiArg(value, what string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", what)
	}
	return n, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
