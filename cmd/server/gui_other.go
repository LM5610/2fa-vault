//go:build !windows

package main

func runDesktopGUI(url string, title string, width, height int) (func(), error) {
	return nil, nil
}
