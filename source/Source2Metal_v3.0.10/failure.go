package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const failExplanationName = "STATUS.txt"

func failDirFor(parent, base string) string {
	return filepath.Join(parent, safeComponent(base))
}

// finalizeFailDir makes a failed/no-output result visible in Explorer without
// flooding the console. Existing diagnostic files are preserved by renaming the
// working result directory and a human-readable explanation is added.
func finalizeFailDir(workDir, parent, base, product, status, reason string, appendReport string) (string, error) {
	finalDir := failDirFor(parent, base)
	sameDir := workDir != "" && strings.EqualFold(filepath.Clean(workDir), filepath.Clean(finalDir))
	if sameDir {
		if err := os.MkdirAll(finalDir, 0755); err != nil {
			return "", err
		}
	} else {
		_ = os.RemoveAll(finalDir)
		if workDir != "" {
			if st, err := os.Stat(workDir); err == nil && st.IsDir() {
				if err := os.Rename(workDir, finalDir); err != nil {
					return "", err
				}
			} else if err := os.MkdirAll(finalDir, 0755); err != nil {
				return "", err
			}
		} else if err := os.MkdirAll(finalDir, 0755); err != nil {
			return "", err
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Source2Metal v%s - %s\n\n", version, product)
	fmt.Fprintf(&b, "Bron          : %s\n", base)
	fmt.Fprintf(&b, "Status        : %s\n", status)
	fmt.Fprintf(&b, "Product       : %s\n\n", product)
	fmt.Fprintln(&b, reason)
	fmt.Fprintln(&b, "\nEr is bewust geen leeg of misleidend productbestand aangemaakt.")
	fmt.Fprintln(&b, "De originele bronbestanden zijn niet gewijzigd.")
	if appendReport != "" {
		if data, err := os.ReadFile(filepath.Join(finalDir, appendReport)); err == nil && len(data) > 0 {
			fmt.Fprintf(&b, "\n----------------------------------------\nDETAILRAPPORT: %s\n----------------------------------------\n\n", appendReport)
			b.Write(data)
			if data[len(data)-1] != '\n' {
				b.WriteByte('\n')
			}
		}
	}
	if err := atomicWriteFile(filepath.Join(finalDir, failExplanationName), []byte(localizeReportText(b.String())), 0644); err != nil {
		return "", err
	}
	return finalDir, nil
}
