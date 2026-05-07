package config

import "testing"

func TestDeepSeekProfileCandidates(t *testing.T) {
	want := map[string][]string{
		"deepseek":          {"--profile", "deepseek", "app-server", "--listen", "stdio://"},
		"deepseek-thinking": {"--profile", "deepseek-thinking", "app-server", "--listen", "stdio://"},
	}

	found := map[string]agentCandidate{}
	for _, c := range agentCandidates {
		if _, ok := want[c.Name]; ok {
			found[c.Name] = c
		}
	}

	for name, args := range want {
		c, ok := found[name]
		if !ok {
			t.Fatalf("missing candidate %q", name)
		}
		if c.Binary != "codex" {
			t.Fatalf("%s Binary = %q, want codex", name, c.Binary)
		}
		if c.Type != "acp" {
			t.Fatalf("%s Type = %q, want acp", name, c.Type)
		}
		if len(c.CheckArgs) != 0 {
			t.Fatalf("%s CheckArgs = %#v, want empty to avoid starting proxy during probe", name, c.CheckArgs)
		}
		if len(c.Args) != len(args) {
			t.Fatalf("%s Args = %#v, want %#v", name, c.Args, args)
		}
		for i := range args {
			if c.Args[i] != args[i] {
				t.Fatalf("%s Args[%d] = %q, want %q; full args=%#v", name, i, c.Args[i], args[i], c.Args)
			}
		}
	}
}
