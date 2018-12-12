/*
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * Author: Liqiang Lau <liqianglau@outlook.com>
 * Site: https://liqiang.io
 */

package main

import (
	"flag"
	"fmt"
	"net/http"
)

var (
	port = 8080
	name = ""
)

func main() {
	flag.StringVar(&name, "n", name, "name for this program")
	flag.IntVar(&port, "p", port, "listen port for this program")
	flag.Parse()

	addr := fmt.Sprintf(":%d", port)
	http.HandleFunc("/", whoIAm)
	if err := http.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}
}

func whoIAm(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("This is %s.", name)
	w.Write([]byte(message))
}
