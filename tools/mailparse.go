package tools

import (
	"regexp"
)

var seenAttachments map[[32]byte]string

// parseText attempts to strip everything out of the text/plain message body except for the latest reply.
//
func parseText(b []byte) ([]byte, error) {

	// Attempt to strip the reply.
	on := regexp.MustCompile(`(?s)On .*? wrote:.*$`)
	b = on.ReplaceAll(b, []byte(""))

	// Remove forwarded-message notice.
	fw := regexp.MustCompile(`(?i)Begin forwarded message:`)
	b = fw.ReplaceAll(b, []byte(""))

	// Eliminate extra whitespace.
	ws := regexp.MustCompile(`(?s)\s{2+}`)
	b = ws.ReplaceAll(b, []byte(" "))

	// Remove reply headers.
	re := regexp.MustCompile(`(.+): ((.|\r\n\s)+)\r\n`)
	b = re.ReplaceAll(b, []byte(""))

	return b, nil
}
