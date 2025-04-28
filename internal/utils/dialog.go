package utils

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SelectFolder opens a folder selection dialog and returns the selected path
func SelectFolder(ctx context.Context) (string, error) {
	return runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{
		Title: "Select Directory",
	})
}

// ShowMessage displays a message dialog with the specified type
func ShowMessage(ctx context.Context, title, message string, dialogType runtime.DialogType) error {
	_, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type:    dialogType,
		Title:   title,
		Message: message,
	})
	return err
}

// ConfirmDialog shows a confirmation dialog and returns the user's choice
func ConfirmDialog(ctx context.Context, title, message string) (bool, error) {
	result, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type:    runtime.QuestionDialog,
		Title:   title,
		Message: message,
		Buttons: []string{"Yes", "No"},
	})

	if err != nil {
		return false, err
	}

	return result == "Yes", nil
}

// SaveFileDialog shows a dialog to select where to save a file
func SaveFileDialog(ctx context.Context, title, defaultFilename string, filters []runtime.FileFilter) (string, error) {
	return runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
		Filters:         filters,
	})
}

// OpenFileDialog shows a dialog to select a file to open
func OpenFileDialog(ctx context.Context, title string, filters []runtime.FileFilter) (string, error) {
	return runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: filters,
	})
}
