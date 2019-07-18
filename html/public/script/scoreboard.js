// Scorebard Alpha
// JavaScript

var interval_all = 15000;
var interval_team = 7500;
var interval_credit = 5000;

function display_close() {
    debug("Displaying closed board...");
    document.write("Closed!");
}
function display_invalid() {
    debug("Displaying invalid board...");
    document.write("Game Not Found");
}

function init() {
    document.sb_auto = false;
    document.sb_debug = true;
    document.sb_loaded = false;
    debug("Starting init.. Selected Game id: " + game);
    if (!game) {
        debug("No game ID detected, bailing..");
        return;
    }
    document.sb_board = document.getElementById("board");
    debug("Opening websocket..");
    document.sb_socket = new WebSocket("ws://" + window.location.host + "/w");
    document.sb_socket.onopen = startup;
    document.sb_socket.onclose = closed;
    document.sb_socket.onmessage = recv;
    debug("Init complete.");
}
function closed() {
    debug("Received websocket close signal." + current);
    if (document.sb_loaded) {
        display_close();
    } else {
        display_invalid();
    }
}
function startup() {
    debug("Received websocket open signal.");
    document.sb_socket.send(JSON.stringify({"game": game}));
}
function auto_scroll() {
    if (!document.sb_auto) {
        return;
    }
    debug("Started auto scroll..");
    var tm = document.getElementById("game-tab").children;
    var nx = false;
    for (var i = 0; i < tm.length; i++) {
        if (tm[i].id === "auto-tab") {
            continue;
        }
        debug(tm[i].id);
        if (nx) {
            auto_set(tm[i], tm);
            return
        }
        nx = tm[i].classList.contains("auto-selected");
        if (nx) {
            tm[i].classList.remove("auto-selected");
        }
    }
    if (tm[0].id !== "auto-tab") {
        auto_set(tm[0], tm);
    } else if (tm[1].id !== "auto-tab") {
        auto_set(tm[1], tm);
    }
}
function recv(message) {
    if (message.data === null && !document.sb_loaded) {
        socket.close()
    } else {
        update_board(message.data);
        if (!document.sb_loaded) {
            navigate("auto");
        }
        document.sb_loaded = true;
    }
}
function update_tabs() {
    var tm = document.getElementsByClassName("team");
    for (var i = 0; i < tm.length; i++) {
        if (tm[i].classList.contains("mini")) {
            continue;
        }
        var ele = document.getElementById(tm[i].id + "-tab");
        if (ele === null) {
            ele = document.createElement("a");
            ele.id = tm[i].id + "-tab";
            var tl = document.getElementById("game-tab");
            if (tl !== null) {
                tl.appendChild(ele);
            }
        }
        var name = document.getElementById(tm[i].id + "-name");
        if (name === null) {
            continue;
        }
        ele.setAttribute("href", "#");
        ele.setAttribute("onclick", "return navigate('" + tm[i].id + "');");
        var si = name.innerHTML.indexOf("<");
        if (si > 0) {
            ele.innerText = name.innerHTML.substr(0, si);
        }
    }
}
function debug(message) {
    if (document.sb_debug) {
        console.log("DEBUG: " + message);
    }
}
function navigate(panel) {
    var tm = document.getElementById("game-tab").children;
    select_div(panel, tm, true);
    if (panel === "auto") {
        var oe = document.getElementById("overview-tab");
        if (oe !== null) {
            document.sb_auto = true;
            oe.classList.add("auto-selected");
            setTimeout(auto_scroll, interval_all);
        }
    } else if (document.sb_auto) {
        document.sb_auto = false;
    }
}
function update_board(data) {
    var up = JSON.parse(data);
    debug("Received " + up.length + " entries...");
    for (var x = 0; x < up.length; x++) {
        handle_update(up[x]);
    }
    update_tabs();
    var gm = document.getElementById("game-status-name");
    document.title = gm.innerText;
}
function auto_set(div, divs) {
    div.classList.add("auto-selected");
    if (div.id === "overview-tab") {
        setTimeout(auto_scroll, interval_all);
    } else if (div.id === "credits-tab") {
        setTimeout(auto_scroll, interval_credit);
    } else {
        setTimeout(auto_scroll, interval_team);
    }
    select_div(div.id.replace("-tab", ""), divs, false);
}
function handle_update(update) {
    debug("Processing '" + JSON.stringify(update) + "'...")
    if (update.remove) {
        var rele = document.getElementById(update.id);
        if (rele !== null) {
            debug("Deleting ID " + update.id + "..");
            rele.remove();
            return
        }
    }
    var pa = null;
    var sr = update.id.lastIndexOf("-");
    if (sr > 0) {
        var pn = update.id.substring(0, sr);
        pa = document.getElementById(pn);
    }
    var ele = document.getElementById(update.id);
    if (ele === null && pa !== null) {
        ele = document.createElement("div")
        ele.id = update.id;
        pa.appendChild(ele);
        debug("Created element " + update.id + "..");
    }
    if (ele === null) {
        return;
    }
    if (update.class) {
        var cl = update.class.split(" ");
        for (var ci = 0; ci < cl.length; ci++) {
            ele.classList.add(cl[ci]);
        }
    }
    if (!update.value) {
        return;
    }
    if (!update.name && update.value !== null) {
        ele.innerText = update.value;
        return;
    }
    if (!update.name) {
        return;
    }
    if (update.name !== "class") {
        ele.style[update.name] = update.value;
        return;
    }
    var cv = update.value[0];
    if (!(cv === "-" || cv === "+")) {
        ele.classList.add(update.value);
        return;
    }
    var cd = update.value.substring(1);
    if (cv == "+") {
        if (!ele.classList.contains(cd)) {
            ele.classList.add(cd);
        }
    }
    if (cv == "-") {
        if (ele.classList.contains(cd)) {
            ele.classList.remove(cd);
        }
    }
}
function select_div(panel, divs, select) {
    for (var i = 0; i < divs.length; i++) {
        name = divs[i].id.replace("-tab", "");
        if (select) {
            if (name === panel) {
                divs[i].classList.add("selected");
            } else {
                divs[i].classList.remove("selected");
                divs[i].classList.remove("auto-selected");
            }
        }
        var pe = document.getElementById(name);
        if (pe !== null) {
            if (name === panel) {
                pe.classList.add("selected");
            } else {
                pe.classList.remove("selected");
            }
        }
    }
    var gt = document.getElementById("game-team");
    if (gt === null) {
        return;
    }
    if (panel === "auto" || panel === "overview") {
        gt.classList.remove("single");
    } else {
        gt.classList.add("single");
    }
}
