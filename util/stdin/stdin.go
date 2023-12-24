package stdin

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
)

// AskListenAddress returns address for server to listen to, taking it from standard input.
func AskListenAddress(log *logrus.Logger) string {
	prompt := "Enter address to listen to in format of 'host:port' or ':port': "
	return ask(log, true, prompt, func(input string) bool {
		return input == ""
	})
}

// ask returns user input, preliminarily printing <prompt>. It runs forever until read is successfull and <callback>
// returns false. If <trim> is true, trim space from user input before passing it to <callback>.
func ask(log *logrus.Logger, trim bool, prompt string, callback func(string) bool) string {
	for {
		fmt.Print(prompt)
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if trim {
			input = strings.TrimSpace(input)
		}
		if callback(input) {
			continue
		}
		if err == nil {
			return input
		}
		log.Error(errors.Wrap(err, "Read from standard input"))
	}
}
