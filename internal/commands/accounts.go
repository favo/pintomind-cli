package commands

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"favo/pintomind-cli/internal/api"
	"favo/pintomind-cli/internal/config"
)

type MeAccount struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	SSOLoginNeeded bool   `json:"sso_login_needed"`
}

type MeNetwork struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	AccountID      int    `json:"account_id"`
	CanManage      bool   `json:"can_manage"`
	SSOLoginNeeded bool   `json:"sso_login_needed"`
}

type MeAccountsResponse struct {
	Accounts []MeAccount `json:"accounts"`
	Networks []MeNetwork `json:"networks"`
}

func fetchMeAccounts(client *api.Client) (*MeAccountsResponse, error) {
	var resp MeAccountsResponse
	if err := client.Get("/me/accounts", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func newMeAccountsCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "accounts",
		Short:       "List the accounts and networks a login may act for (OAuth connections)",
		Annotations: map[string]string{NoAccountAnnotation: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := app(cmd)
			client := *a.Client
			client.AccountID, client.NetworkID = 0, 0
			resp, err := fetchMeAccounts(&client)
			if err != nil {
				return err
			}
			if a.JSONOutput {
				printJSON(resp)
				return nil
			}
			d := a.Config.Domains[a.ActiveConnection]
			printAccountChoices(cmd, resp, d)
			return nil
		},
	}
}

func printAccountChoices(cmd *cobra.Command, resp *MeAccountsResponse, current config.Domain) {
	rows := make([][]string, 0, len(resp.Accounts)+len(resp.Networks))
	for _, acc := range resp.Accounts {
		rows = append(rows, []string{
			marker(current.NetworkID == 0 && current.AccountID == acc.ID),
			"account", strconv.Itoa(acc.ID), acc.Name, acc.Role, "yes", ssoNote(acc.SSOLoginNeeded),
		})
	}
	for _, n := range resp.Networks {
		rows = append(rows, []string{
			marker(current.NetworkID == n.ID),
			"network", strconv.Itoa(n.ID), n.Name, n.Role, yesNo(n.CanManage), ssoNote(n.SSOLoginNeeded),
		})
	}
	printTable(cmd, []string{"", "KIND", "ID", "NAME", "ROLE", "CAN WRITE", "SSO"}, rows)
}

func marker(active bool) string {
	if active {
		return "*"
	}
	return ""
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "read only"
}

func ssoNote(needed bool) string {
	if needed {
		return "log in via SSO"
	}
	return ""
}

type accountChoice struct {
	network bool
	id      int
	name    string
}

func (c accountChoice) apply(d *config.Domain) {
	d.AccountID, d.AccountName, d.NetworkID, d.NetworkName = 0, "", 0, ""
	if c.network {
		d.NetworkID, d.NetworkName = c.id, c.name
	} else {
		d.AccountID, d.AccountName = c.id, c.name
	}
}

func findAccount(resp *MeAccountsResponse, id int) (accountChoice, error) {
	for _, acc := range resp.Accounts {
		if acc.ID == id {
			if acc.SSOLoginNeeded {
				fmt.Fprintf(os.Stderr, "Note: account %s requires SSO; log in again through its SSO before using it.\n", acc.Name)
			}
			return accountChoice{id: acc.ID, name: acc.Name}, nil
		}
	}
	return accountChoice{}, fmt.Errorf("account %d is not one this login may act for (see: pintomind me accounts)", id)
}

func findNetwork(resp *MeAccountsResponse, id int) (accountChoice, error) {
	for _, n := range resp.Networks {
		if n.ID == id {
			if !n.CanManage {
				fmt.Fprintf(os.Stderr, "Note: your role may only read network %s, not change it.\n", n.Name)
			}
			if n.SSOLoginNeeded {
				fmt.Fprintf(os.Stderr, "Note: network %s requires SSO; log in again through its SSO before using it.\n", n.Name)
			}
			return accountChoice{network: true, id: n.ID, name: n.Name}, nil
		}
	}
	return accountChoice{}, fmt.Errorf("network %d is not one this login may act for (see: pintomind me accounts)", id)
}

var errNoChoice = errors.New("no account chosen")

// pickAccount lets the user choose what a login acts for, among accounts, networks or both
// (kind ""). A single choice is picked without asking; without a terminal it gives up with
// errNoChoice.
func pickAccount(resp *MeAccountsResponse, current config.Domain, kind string) (accountChoice, error) {
	var options []huh.Option[accountChoice]
	if kind != "network" {
		for _, acc := range resp.Accounts {
			label := fmt.Sprintf("%s (account %d, %s)", acc.Name, acc.ID, acc.Role)
			if acc.SSOLoginNeeded {
				label += " — needs SSO login"
			}
			options = append(options, huh.NewOption(label, accountChoice{id: acc.ID, name: acc.Name}).
				Selected(current.NetworkID == 0 && current.AccountID == acc.ID))
		}
	}
	if kind != "account" {
		for _, n := range resp.Networks {
			label := fmt.Sprintf("%s (network %d, %s)", n.Name, n.ID, n.Role)
			if !n.CanManage {
				label += " — read only"
			}
			options = append(options, huh.NewOption(label, accountChoice{network: true, id: n.ID, name: n.Name}).
				Selected(current.NetworkID == n.ID))
		}
	}

	what := map[string]string{"": "account or network", "account": "account", "network": "network"}[kind]
	switch {
	case len(options) == 0:
		return accountChoice{}, fmt.Errorf("this login may act for no %s", what)
	case len(options) == 1:
		return options[0].Value, nil
	case !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()):
		return accountChoice{}, errNoChoice
	}

	var choice accountChoice
	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[accountChoice]().Title("Act for which "+what+"?").Options(options...).Value(&choice),
	)).Run()
	return choice, err
}

func saveChoice(name string, choice accountChoice) error {
	_, err := config.Update(func(cfg *config.Config) error {
		d, ok := cfg.Domains[name]
		if !ok {
			return fmt.Errorf("connection %q not found", name)
		}
		choice.apply(&d)
		cfg.Domains[name] = d
		return nil
	})
	return err
}
