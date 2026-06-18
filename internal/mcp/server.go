package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kurt_v1/internal/think"
)

// Cfg controls the MCP server behaviour.
type Cfg struct {
	Workdir         string // default: $PWD at server start
	ShellTimeoutSec int    // default: 30
	ProviderCfg     think.ProviderConfig
}

// Serve starts the MCP server over stdio. Blocks until stdin closes.
func Serve(cfg Cfg) error {
	if cfg.Workdir == "" {
		cfg.Workdir, _ = os.Getwd()
	}
	if cfg.ShellTimeoutSec <= 0 {
		cfg.ShellTimeoutSec = 30
	}

	s := server.NewMCPServer("kurt", "0.1.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	addShellExec(s, cfg)
	addReadFile(s, cfg)
	addWriteFile(s, cfg)
	addListDirectory(s, cfg)
	addGitContext(s, cfg)
	addThink(s, cfg)

	addShellResource(s, cfg)
	addGitResource(s, cfg)

	return server.ServeStdio(s)
}

// ── tools ─────────────────────────────────────────────────────────────────────

func addShellExec(s *server.MCPServer, cfg Cfg) {
	s.AddTool(mcp.NewTool("shell_exec",
		mcp.WithDescription("Run a shell command in the working directory. Returns combined stdout+stderr."),
		mcp.WithString("command", mcp.Required(), mcp.Description("Shell command to execute")),
		mcp.WithNumber("timeout_sec", mcp.Description("Timeout in seconds (default: 30)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cmd := mcp.ParseString(req, "command", "")
		if cmd == "" {
			return mcp.NewToolResultError("command is required"), nil
		}
		secs := mcp.ParseInt(req, "timeout_sec", cfg.ShellTimeoutSec)
		tctx, cancel := context.WithTimeout(ctx, time.Duration(secs)*time.Second)
		defer cancel()

		c := exec.CommandContext(tctx, "sh", "-c", cmd)
		c.Dir = cfg.Workdir
		out, err := c.CombinedOutput()
		text := strings.TrimRight(string(out), "\n")
		if err != nil {
			if text != "" {
				return mcp.NewToolResultText(text + "\n[exit: " + err.Error() + "]"), nil
			}
			return mcp.NewToolResultError(err.Error()), nil
		}
		if text == "" {
			text = "(no output)"
		}
		return mcp.NewToolResultText(text), nil
	})
}

func addReadFile(s *server.MCPServer, cfg Cfg) {
	s.AddTool(mcp.NewTool("read_file",
		mcp.WithDescription("Read the contents of a file. Relative paths are resolved from the working directory."),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to read")),
		mcp.WithNumber("max_bytes", mcp.Description("Maximum bytes to read (default: 65536)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw := mcp.ParseString(req, "path", "")
		if raw == "" {
			return mcp.NewToolResultError("path is required"), nil
		}
		p := resolvePath(cfg.Workdir, raw)
		maxBytes := mcp.ParseInt(req, "max_bytes", 65536)

		f, err := os.Open(p)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		defer f.Close()
		buf := make([]byte, maxBytes)
		n, _ := f.Read(buf)
		return mcp.NewToolResultText(string(buf[:n])), nil
	})
}

func addWriteFile(s *server.MCPServer, cfg Cfg) {
	s.AddTool(mcp.NewTool("write_file",
		mcp.WithDescription("Write content to a file. Creates parent directories as needed."),
		mcp.WithString("path", mcp.Required(), mcp.Description("File path to write")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to write")),
		mcp.WithBoolean("append", mcp.Description("Append instead of overwrite (default: false)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw := mcp.ParseString(req, "path", "")
		if raw == "" {
			return mcp.NewToolResultError("path is required"), nil
		}
		p := resolvePath(cfg.Workdir, raw)
		content := mcp.ParseString(req, "content", "")
		doAppend := mcp.ParseBoolean(req, "append", false)

		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		flag := os.O_WRONLY | os.O_CREATE
		if doAppend {
			flag |= os.O_APPEND
		} else {
			flag |= os.O_TRUNC
		}
		f, err := os.OpenFile(p, flag, 0644)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		defer f.Close()
		if _, err := f.WriteString(content); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("wrote %d bytes to %s", len(content), p)), nil
	})
}

func addListDirectory(s *server.MCPServer, cfg Cfg) {
	s.AddTool(mcp.NewTool("list_directory",
		mcp.WithDescription("List files and directories. Returns a newline-separated list of paths."),
		mcp.WithString("path", mcp.Description("Directory path (default: working directory)")),
		mcp.WithBoolean("recursive", mcp.Description("Walk recursively (default: false)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw := mcp.ParseString(req, "path", "")
		var p string
		if raw == "" {
			p = cfg.Workdir
		} else {
			p = resolvePath(cfg.Workdir, raw)
		}
		recursive := mcp.ParseBoolean(req, "recursive", false)

		var lines []string
		if recursive {
			_ = filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				rel, _ := filepath.Rel(p, path)
				if rel == "." {
					return nil
				}
				if info.IsDir() {
					lines = append(lines, rel+"/")
				} else {
					lines = append(lines, rel)
				}
				return nil
			})
		} else {
			entries, err := os.ReadDir(p)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			for _, e := range entries {
				name := e.Name()
				if e.IsDir() {
					name += "/"
				}
				lines = append(lines, name)
			}
		}

		if len(lines) == 0 {
			return mcp.NewToolResultText("(empty directory)"), nil
		}
		return mcp.NewToolResultText(strings.Join(lines, "\n")), nil
	})
}

func addGitContext(s *server.MCPServer, cfg Cfg) {
	s.AddTool(mcp.NewTool("git_context",
		mcp.WithDescription("Get current git state: branch, dirty flag, recent commits, and working tree status."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(buildGitSummary(cfg.Workdir)), nil
	})
}

func addThink(s *server.MCPServer, cfg Cfg) {
	s.AddTool(mcp.NewTool("think",
		mcp.WithDescription("Ask kurt's configured LLM with full shell and git context injected automatically."),
		mcp.WithString("question", mcp.Required(), mcp.Description("The question or task for the AI")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q := mcp.ParseString(req, "question", "")
		if q == "" {
			return mcp.NewToolResultError("question is required"), nil
		}
		p, err := think.New(cfg.ProviderCfg)
		if err != nil {
			return mcp.NewToolResultError("provider: " + err.Error()), nil
		}
		thinkCtx := think.Context{
			CWD:         cfg.Workdir,
			Git:         think.CollectGit(cfg.Workdir),
			ProjectType: think.CollectProjectType(cfg.Workdir),
			GitLog:      think.CollectGitLog(cfg.Workdir, 8),
			Env:         think.CollectEnv(),
		}
		var buf bytes.Buffer
		if err := p.ThinkStream(thinkCtx, q, &buf); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(buf.String()), nil
	})
}

// ── resources ─────────────────────────────────────────────────────────────────

func addShellResource(s *server.MCPServer, cfg Cfg) {
	s.AddResource(
		mcp.NewResource("kurt://context/shell", "Shell context",
			mcp.WithMIMEType("text/plain"),
			mcp.WithResourceDescription("Working directory, project type, and relevant env vars"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			var b strings.Builder
			b.WriteString("cwd: " + cfg.Workdir + "\n")
			if pt := think.CollectProjectType(cfg.Workdir); pt != "" {
				b.WriteString("project: " + pt + "\n")
			}
			if env := think.CollectEnv(); len(env) > 0 {
				b.WriteString("\nenv:\n")
				for k, v := range env {
					b.WriteString("  " + k + "=" + v + "\n")
				}
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{URI: "kurt://context/shell", MIMEType: "text/plain", Text: b.String()},
			}, nil
		},
	)
}

func addGitResource(s *server.MCPServer, cfg Cfg) {
	s.AddResource(
		mcp.NewResource("kurt://context/git", "Git context",
			mcp.WithMIMEType("text/plain"),
			mcp.WithResourceDescription("Current git branch, status, and recent commits"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{URI: "kurt://context/git", MIMEType: "text/plain", Text: buildGitSummary(cfg.Workdir)},
			}, nil
		},
	)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func buildGitSummary(cwd string) string {
	git := think.CollectGit(cwd)
	if git == nil {
		return "(not a git repository)"
	}
	var b strings.Builder
	b.WriteString("branch: " + git.Branch + "\n")
	if git.Dirty {
		b.WriteString("dirty: true\n")
	}
	if logs := think.CollectGitLog(cwd, 10); len(logs) > 0 {
		b.WriteString("\nrecent commits:\n")
		for _, l := range logs {
			b.WriteString("  " + l + "\n")
		}
	}
	out, err := exec.Command("git", "-C", cwd, "status", "--short").Output()
	if err == nil && len(out) > 0 {
		b.WriteString("\nstatus:\n")
		b.Write(out)
	}
	return b.String()
}

func resolvePath(workdir, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(workdir, p)
}
