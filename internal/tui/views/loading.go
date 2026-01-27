package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderLoadingBanner renders a centered loading indicator with spinner frame.
func RenderLoadingBanner(spinnerFrame, message string, width int) string {
	content := LoadingSpinnerStyle.Render(spinnerFrame) + " " + LoadingMsgStyle.Render(message)
	contentWidth := lipgloss.Width(content)
	padding := (width - contentWidth) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + content
}

// RenderLoadingInline renders an inline loading indicator with spinner frame.
func RenderLoadingInline(spinnerFrame, message string) string {
	return LoadingSpinnerStyle.Render(spinnerFrame) + " " + LoadingMsgStyle.Render(message)
}
