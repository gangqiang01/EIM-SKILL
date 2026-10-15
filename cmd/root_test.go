package cmd

import "testing"

func TestRootFlagSnapshotPreservesStartupConfig(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("config flag not registered")
	}

	originalValue := flag.Value.String()
	originalChanged := flag.Changed
	t.Cleanup(func() {
		_ = flag.Value.Set(originalValue)
		flag.Changed = originalChanged
	})

	custom := "/tmp/custom-config.yaml"
	if err := flag.Value.Set(custom); err != nil {
		t.Fatalf("set startup config flag: %v", err)
	}
	flag.Changed = true

	snapshot := snapshotFlagState(rootCmd)
	resetCommandTree(rootCmd)
	restoreFlagState(rootCmd, snapshot)

	if got := cfgFile; got != custom {
		t.Fatalf("cfgFile after restore = %q, want %q", got, custom)
	}
	if !flag.Changed {
		t.Fatal("config flag should remain changed after restore")
	}
}
