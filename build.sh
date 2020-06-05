#!/usr/bin/bash
# Copyright (C) 2020 iDigitalFlame
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published
# by the Free Software Foundation, either version 3 of the License, or
# any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.
#

output="../bin/scoreboard"
if [ $# -ge 1 ]; then
    output="$1"
fi

which packr2 &> /dev/null
if [ $? -ne 0 ]; then
    printf "Installing packr..\n"
    bash -c "cd scoreboard; go get -u github.com/gobuffalo/packr/v2/packr2"
fi

packr=$(which packr2)
printf "Building...\n"
bash -c "cd scoreboard; $packr"
cat <<EOF > scoreboard/scoreboard-packr.go
// +build !skippackr
package scoreboard

import _ "github.com/iDigitalFlame/scorebot-scoreboard/scoreboard/packrd"
EOF
bash -c "cd scoreboard; go build -trimpath -ldflags '-s -w' -o '$output' cmd/main.go; $packr clean"

which upx &> /dev/null
if [ $? -eq 0 ] && [ -f "$output" ]; then
    upx --compress-exports=1 --strip-relocs=1 --compress-icons=2 --best --no-backup -9 "$output"
fi

printf "Done!\n"
