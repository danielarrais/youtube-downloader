package main

import "testing"

func TestCleanYouTubeURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "removes video parameters",
			url:  "https://www.youtube.com/watch?v=3CWL9WXYSWU&list=RD3CWL9WXYSWU&start_radio=1",
			want: "https://www.youtube.com/watch?v=3CWL9WXYSWU",
		},
		{
			name: "keeps playlist identifier",
			url:  "https://www.youtube.com/playlist?list=PLrzuA2--JVdSPr5FdYtBixGMw9uCWfsSi&index=2",
			want: "https://www.youtube.com/playlist?list=PLrzuA2--JVdSPr5FdYtBixGMw9uCWfsSi",
		},
		{
			name: "trims spaces",
			url:  "  https://youtu.be/3CWL9WXYSWU  ",
			want: "https://youtu.be/3CWL9WXYSWU",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := cleanYouTubeURL(test.url); got != test.want {
				t.Fatalf("cleanYouTubeURL(%q) = %q, want %q", test.url, got, test.want)
			}
		})
	}
}

func TestSaveConfigDefaultsInvalidQuality(t *testing.T) {
	app := NewApp()
	originalConfig := globalConfig
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(func() {
		globalConfig = originalConfig
	})

	globalConfig = Config{DownloadDir: t.TempDir(), Quality: "192k"}
	config := app.SaveConfig(Config{
		DownloadDir: t.TempDir(),
		Quality:     "invalid",
	})

	if config.Quality != "192k" {
		t.Fatalf("SaveConfig() quality = %q, want %q", config.Quality, "192k")
	}
}
