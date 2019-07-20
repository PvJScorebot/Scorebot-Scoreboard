//
//  Scoreboard v1: Alpha
//  iDigitalFlame 2019, The Scorebot Project.
//
//  Javascript Main
//

// Auto Interval Constants
var interval_all = 15000;
var interval_team = 7500;
var interval_credit = 5000;

function init() {
    document.sb_auto = false;
    document.sb_debug = true;
    document.sb_loaded = false;
    debug("Starting init.. Selected Game id: " + game);
    if (!game) {
        debug("No game ID detected, bailing..");
        return;
    }
    document.sb_event = document.getElementById("event");
    document.sb_board = document.getElementById("board");
    setInterval(scroll_beacons, 200);
    debug("Opening websocket..");
    document.sb_socket = new WebSocket("ws://" + window.location.host + "/w");
    document.sb_socket.onopen = startup;
    document.sb_socket.onclose = closed;
    document.sb_socket.onmessage = recv;
    debug("Init complete.");
}
function closed() {
    debug("Received websocket close signal.");
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
function event_close() {
    document.sb_event.style.display = "none";
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
    var oe = document.getElementById("overview-tab");
    var tm = document.getElementById("game-tab").children;
    select_div(panel, tm, true);
    if (panel === "auto") {
        if (oe !== null) {
            document.sb_auto = true;
            oe.classList.add("auto-selected");
            setTimeout(auto_scroll, interval_all);
        }
    } else if (document.sb_auto) {
        var oe = document.getElementById("overview-tab");
        if (oe !== null) {
            oe.classList.remove("auto-selected");
        }
        document.sb_auto = false;
    }
}
function display_close() {
    debug("Displaying closed board...");
    var em = document.getElementById("game-disconnected");
    if (em !== null) {
        em.style.display = "block";
    }
}
function scroll_beacons() {
    var bt = document.getElementsByClassName("team-beacon");
    for (var bi = 0; bi < bt.length; bi++) {
        scroll_beacon(bt[bi]);
    }
}
function update_beacons() {
    var bl = document.getElementsByClassName("beacon");
    for (var bi = 0; bi < bl.length; bi++) {
        set_beacon_image(bl[bi]);
    }
}
function display_invalid() {
    debug("Displaying invalid board...");
    var em = document.getElementById("game-invalid");
    if (em !== null) {
        em.style.display = "block";
    }
}
function update_board(data) {
    var up = JSON.parse(data);
    debug("Received " + up.length + " entries...");
    for (var x = 0; x < up.length; x++) {
        handle_update(up[x]);
    }
    update_tabs();
    update_beacons();
    var gm = document.getElementById("game-status-name");
    if (gm !== null) {
        document.title = gm.innerText;
    }
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
function scroll_beacon(beacon) {
    var bc = beacon.children[0];
    if (bc === null) {
        return;
    }
    if (bc.offsetHeight <= beacon.offsetHeight) {
        beacon.classList.remove("reverse");
        return;
    }
    var rev = beacon.classList.contains("reverse");
    if ((beacon.scrollTop + beacon.offsetHeight) == bc.offsetHeight) {
        beacon.classList.add("reverse");
        rev = true;
    } else if (beacon.scrollTop == 0) {
        beacon.classList.remove("reverse");
        rev = false;
    }
    if (rev) {
        beacon.scrollTop -= 5;
    } else {
        beacon.scrollTop += 5;
    }
}
function set_beacon_image(beacon) {
    if (beacon.style.background.indexOf("data") >= 0) {
        return
    }
    var bg = beacon.style.background;
    if (bg === null || bg === "") {
        return;
    }
    var color = [0, 0, 0];
    if (bg.indexOf(",") > 0) {
        var bs = bg.split(",");
        if (bs.length == 3) {
            color[0] = parseInt(bs[0].replace(")", "").replace("rgb(", ""), 10);
            color[1] = parseInt(bs[1].replace(")", "").replace("rgb(", ""), 10);
            color[2] = parseInt(bs[2].replace(")", "").replace("rgb(", ""), 10);
        }
    } else {
        var bs = hex.match(/[a-f0-9]{2}/gi);
        if (bs.length == 3) {
            color[0] = parseInt(bs[0], 16);
            color[1] = parseInt(bs[1], 16);
            color[2] = parseInt(bs[2], 16);
        }
    }
    var image = new Image();
    image.onload = function () {
        var canvas = document.createElement("canvas");
        canvas.width = 25;
        canvas.height = 25;
        var layer = canvas.getContext("2d");
        layer.drawImage(this, 0, 0);
        var draw = layer.getImageData(0, 0, 128, 128);
        for (var i = 0; i < draw.data.length; i += 4) {
            draw.data[i] = color[0];
            draw.data[i + 1] = color[1];
            draw.data[i + 2] = color[2];
        }
        layer.putImageData(draw, 0, 0);
        beacon.style.background = "url('" + canvas.toDataURL("image/png") + "')";
    }
    image.src = "/image/beacon.png";
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

function show_event(event) {

}
function show_event_popup(message) {

}