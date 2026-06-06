package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kkdai/youtube/v2"
)

func TestGetClientReturnsIsolatedClients(t *testing.T) {
	if getClient() == getClient() {
		t.Fatal("getClient() reused a client with mutable YouTube state")
	}
}

func TestFormatYouTubeUnavailableError(t *testing.T) {
	err := &youtube.ErrPlayabiltyStatus{
		Status: "ERROR",
		Reason: "This video is unavailable",
	}

	message := FormatYouTubeError(err, "pt-BR")
	if message != "O YouTube informou que este vídeo está indisponível. Ele pode ter sido removido, tornado privado ou bloqueado para esta região." {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestFormatYouTubeOtherPlaybackErrorKeepsDetails(t *testing.T) {
	err := errors.New("playback failed")
	message := FormatYouTubeError(err, "pt-BR")
	if message != "Erro ao consultar o YouTube: playback failed" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestFormatAgeRestrictionError(t *testing.T) {
	err := fmt.Errorf("can't bypass age restriction: %w", youtube.ErrNotPlayableInEmbed)

	portuguese := FormatYouTubeError(err, "pt-BR")
	if portuguese != "Este vídeo possui uma restrição de idade ou reprodução que impede o download sem login." {
		t.Fatalf("unexpected Portuguese message: %q", portuguese)
	}

	english := FormatYouTubeError(err, "en-US")
	if english != "This video has an age or playback restriction that prevents downloading without signing in." {
		t.Fatalf("unexpected English message: %q", english)
	}
}

func TestTranslateStoredAgeRestrictionError(t *testing.T) {
	oldMessage := "Erro ao consultar o YouTube: can't bypass age restriction: embedding of this video has been disabled"

	message := TranslateStoredYouTubeError(oldMessage, "pt-BR")
	if message != "Este vídeo possui uma restrição de idade ou reprodução que impede o download sem login." {
		t.Fatalf("unexpected translated message: %q", message)
	}
}
