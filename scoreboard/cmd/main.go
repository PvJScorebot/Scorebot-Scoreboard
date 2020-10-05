// Copyright(C) 2020 iDigitalFlame
//
// This program is free software: you can redistribute it and / or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.If not, see <https://www.gnu.org/licenses/>.
//

package main

import (
	"flag"
	"os"

	"github.com/iDigitalFlame/scorebot-scoreboard/scoreboard"
)

func main() {
	s, err := scoreboard.Cmdline()
	if err == flag.ErrHelp {
		os.Exit(2)
	}

	if err != nil {
		os.Stderr.WriteString("Error during startup: " + err.Error() + "!\n")
		os.Exit(1)
	}

	if s == nil {
		os.Exit(0)
	}

	if err := s.Run(); err != nil {
		os.Stderr.WriteString("Error during runtime: " + err.Error() + "!\n")
		os.Exit(1)
	}
}
