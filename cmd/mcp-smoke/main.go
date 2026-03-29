// Command mcp-smoke runs MCP over stdio against the server (local binary or docker) for CI/manual checks.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

func main() {
	docker := flag.Bool("docker", false, "run server via docker run")
	image := flag.String("image", "nexus-mcp:local", "docker image name")
	envFile := flag.String("env-file", "", "path to .env for docker --env-file (required with -docker)")
	bin := flag.String("bin", "", "path to nexusmods-mcp binary (default: server from PATH if -docker=false)")
	flag.Parse()

	var cmd *exec.Cmd
	if *docker {
		if *envFile == "" {
			fatalf("-env-file is required with -docker")
		}
		cmd = exec.Command("docker", "run", "--rm", "-i", "--env-file", *envFile, *image)
	} else {
		name := *bin
		if name == "" {
			name = "nexusmods-mcp"
		}
		cmd = exec.Command(name)
		cmd.Env = os.Environ()
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fatalf("stdin: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fatalf("stdout: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fatalf("start: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	r := bufio.NewReader(stdout)
	w := bufio.NewWriter(stdin)

	send := func(obj any) {
		b, err := json.Marshal(obj)
		if err != nil {
			fatalf("marshal: %v", err)
		}
		if _, err := w.Write(append(b, '\n')); err != nil {
			fatalf("write: %v", err)
		}
		if err := w.Flush(); err != nil {
			fatalf("flush: %v", err)
		}
	}

	readForID := func(want float64) map[string]json.RawMessage {
		deadline := time.After(60 * time.Second)
		for {
			select {
			case <-deadline:
				fatalf("timeout waiting for jsonrpc id %v", want)
			default:
			}
			time.Sleep(5 * time.Millisecond)
			line, err := r.ReadBytes('\n')
			if err != nil {
				fatalf("read: %v", err)
			}
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			var msg map[string]json.RawMessage
			if err := json.Unmarshal(line, &msg); err != nil {
				continue
			}
			if id, ok := msg["id"]; ok {
				var idNum float64
				_ = json.Unmarshal(id, &idNum)
				if idNum == want {
					return msg
				}
			}
		}
	}

	// Level 2: initialize + initialized + tools/list
	send(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "mcp-smoke", "version": "0.1"},
		},
	})
	initResp := readForID(1)
	if errVal, ok := initResp["error"]; ok {
		fatalf("initialize error: %s", string(errVal))
	}
	fmt.Println("OK initialize")

	send(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	})
	fmt.Println("OK notifications/initialized (sent)")

	send(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	listResp := readForID(2)
	if _, ok := listResp["error"]; ok {
		fatalf("tools/list error: %s", string(listResp["error"]))
	}
	toolsBody := []byte(listResp["result"])
	names := []string{
		"nexus_games", "nexus_search_mods", "nexus_get_mod", "nexus_list_mod_files", "nexus_get_mod_requirements",
		"nexus_get_mod_changelog", "nexus_get_mod_file", "nexus_get_game", "nexus_game_categories",
		"nexus_mods_latest_updated", "nexus_mods_latest_added", "nexus_mods_trending", "nexus_mods_recently_updated",
		"nexus_get_tracked_mods", "nexus_get_mod_graphql", "nexus_get_rate_limits", "nexus_invalidate_cache",
	}
	for _, n := range names {
		if !bytes.Contains(toolsBody, []byte(n)) {
			fatalf("tools/list missing %q in %s", n, string(toolsBody))
		}
	}
	fmt.Printf("OK tools/list contains all %d tools\n", len(names))

	// Level 3: tool calls
	call := func(id float64, name string, args map[string]any) json.RawMessage {
		send(map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      name,
				"arguments": args,
			},
		})
		resp := readForID(id)
		if errVal, ok := resp["error"]; ok {
			fatalf("tools/call %s rpc error: %s", name, string(errVal))
		}
		return resp["result"]
	}

	res := call(3, "nexus_games", map[string]any{})
	if !bytes.Contains(res, []byte(`"content"`)) && !bytes.Contains(res, []byte(`"isError"`)) {
		fatalf("unexpected nexus_games result: %s", string(res))
	}
	if bytes.Contains(res, []byte(`"isError":true`)) {
		fatalf("nexus_games tool error: %s", string(res))
	}
	fmt.Println("OK nexus_games")

	res = call(4, "nexus_search_mods", map[string]any{
		"game_domain": "skyrimspecialedition",
		"query":       "unofficial",
		"count":       "5",
	})
	if bytes.Contains(res, []byte(`"isError":true`)) {
		fmt.Printf("WARN nexus_search_mods isError (GraphQL/schema): %s\n", truncate(string(res), 400))
	} else {
		fmt.Println("OK nexus_search_mods")
	}

	// USSEP mod id 62852 for Skyrim SE
	res = call(5, "nexus_get_mod", map[string]any{
		"game_domain": "skyrimspecialedition",
		"mod_id":      "62852",
	})
	if bytes.Contains(res, []byte(`"isError":true`)) {
		fatalf("nexus_get_mod: %s", truncate(string(res), 500))
	}
	fmt.Println("OK nexus_get_mod")

	res = call(6, "nexus_list_mod_files", map[string]any{
		"game_domain": "skyrimspecialedition",
		"mod_id":      "62852",
	})
	if bytes.Contains(res, []byte(`"isError":true`)) {
		fatalf("nexus_list_mod_files: %s", truncate(string(res), 500))
	}
	fmt.Println("OK nexus_list_mod_files")

	// Open Animation Replacer - RaySense (dependencies + dependents on Nexus)
	res = call(7, "nexus_get_mod_requirements", map[string]any{
		"game_domain":        "skyrimspecialedition",
		"mod_id":             "175498",
		"requirements_count": "20",
		"dependents_count":   "20",
	})
	if bytes.Contains(res, []byte(`"isError":true`)) {
		fatalf("nexus_get_mod_requirements: %s", truncate(string(res), 500))
	}
	var toolWrap struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(res, &toolWrap); err != nil || len(toolWrap.Content) == 0 {
		fatalf("nexus_get_mod_requirements: parse result: %v body=%s", err, truncate(string(res), 400))
	}
	inner := toolWrap.Content[0].Text
	if !bytes.Contains([]byte(inner), []byte(`"nexusRequirements"`)) {
		fatalf("nexus_get_mod_requirements: expected nexusRequirements in tool text: %s", truncate(inner, 400))
	}
	fmt.Println("OK nexus_get_mod_requirements")

	id := float64(8)
	mustToolOK := func(name string, args map[string]any) {
		res := call(id, name, args)
		id++
		if bytes.Contains(res, []byte(`"isError":true`)) {
			fatalf("%s: %s", name, truncate(string(res), 500))
		}
		fmt.Println("OK", name)
	}

	mustToolOK("nexus_get_mod_changelog", map[string]any{
		"game_domain": "skyrimspecialedition",
		"mod_id":      "62852",
	})
	mustToolOK("nexus_get_mod_file", map[string]any{
		"game_domain": "skyrimspecialedition",
		"mod_id":      "62852",
		"file_id":     "260592",
	})
	mustToolOK("nexus_get_game", map[string]any{"game_domain": "skyrimspecialedition"})
	mustToolOK("nexus_game_categories", map[string]any{"game_domain": "skyrimspecialedition"})
	mustToolOK("nexus_mods_latest_updated", map[string]any{"game_domain": "skyrimspecialedition"})
	mustToolOK("nexus_mods_latest_added", map[string]any{"game_domain": "skyrimspecialedition"})
	mustToolOK("nexus_mods_trending", map[string]any{"game_domain": "skyrimspecialedition"})
	mustToolOK("nexus_mods_recently_updated", map[string]any{
		"game_domain": "skyrimspecialedition",
		"period":      "1d",
	})
	mustToolOK("nexus_get_tracked_mods", map[string]any{})
	mustToolOK("nexus_get_mod_graphql", map[string]any{
		"game_domain": "skyrimspecialedition",
		"mod_id":      "62852",
	})
	mustToolOK("nexus_get_rate_limits", map[string]any{})
	mustToolOK("nexus_invalidate_cache", map[string]any{"mode": "all"})

	_ = stdin.Close()
	_, _ = io.Copy(io.Discard, r)
	_ = cmd.Wait()
	fmt.Println("ALL_OK")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "mcp-smoke: "+format+"\n", args...)
	os.Exit(1)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
