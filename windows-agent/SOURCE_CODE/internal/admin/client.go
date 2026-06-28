package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/nossh/nossh/internal/registry"
)

type Client struct {
	BaseURL string
	Token   string
}

func (c *Client) List() ([]registry.AgentRecord, error) {
	var out []registry.AgentRecord
	if err := c.get("/api/agents", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) AssignName(code, name string) (*registry.AgentRecord, error) {
	var out registry.AgentRecord
	err := c.post("/api/agents/"+code+"/name", map[string]string{"name": name}, &out)
	return &out, err
}

func (c *Client) Delete(code string) error {
	return c.delete("/api/agents/" + code)
}

func (c *Client) Rename(name, newName string) (*registry.AgentRecord, error) {
	var out registry.AgentRecord
	err := c.post("/api/rename", map[string]string{"name": name, "new_name": newName}, &out)
	return &out, err
}

func (c *Client) Revoke(name string) error {
	return c.post("/api/revoke", map[string]string{"name": name}, nil)
}

func PrintList(agents []registry.AgentRecord) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CODE\tHOSTNAME\tNAME\tSTATUS\tONLINE\tLAST SEEN")
	for _, a := range agents {
		name := a.Name
		if name == "" {
			name = "—"
		}
		online := "no"
		if a.Online {
			online = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			a.Code, a.Hostname, name, a.Status, online, a.LastSeen.Format(time.RFC3339))
	}
	_ = w.Flush()
}

func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	c.applyAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func (c *Client) post(path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func (c *Client) delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	c.applyAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s", strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) applyAuth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

func decode(resp *http.Response, out any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s", strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}
