// Copyright(C) 2020 - 2023 iDigitalFlame
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
//  Scoreboard v2.3
//  2020 iDigitalFlame
//
//  Javascript Main File
//

// Close messages
const messages = [
    "Whatda trying to pull here?",
    "Delete system32 to speed up your computer!",
    "Loading reverse shell...",
    "HACK THE PLANET!",
    "The cake is a lie!",
    "This statement is false!",
    "Hey buddy, hands off!",
    "I'm sorry Dave, I cannot do that.",
    "This is no escape, only Zuul.",
    "No one expects the Spanish Inquisition!",
    "WARNING: Not intended for indoor use.",
    "E = mc^(OMG)/wtf",
    "Don't make me hack you scrub!",
    "I found your IP address '127.0.0.1'! prepare to get DDoSd noob!",
    "Proving P = NP...",
    "Computing 6 x 9...",
    "Mining bitcoin...",
    "Dividing by 0...",
    "Initializing Skynet...",
    "[REDACTED]",
    "Downloading more RAM...",
    "Ordering 1s and 0s...",
    "Navigating neural network...",
    "Importing machine learning...",
    "Issuing Alice and Bob one-time pads...",
    "Mining bitcoin cash...",
    "Symlinking emacs and vim to ed...",
    "Generating key material by trying to escape vim...",
    "(creating unresolved tension...",
];

// Auto Interval Constants
const interval_all = 15000;
const interval_team = 7500;
const interval_credit = 5000;

function init() {
    document.sb_auto = false;
    document.sb_loaded = false;
    document.sb_callout = false;
    document.sb_tab_offset = null;
    document.sb_debug = document.location.toString().indexOf("?debug") > 0;
    debug("Starting init.. Selected Game id: " + game);
    if (!game) {
        debug("No game ID detected, bailing!");
        return;
    }
    window.addEventListener("resize", check_mobile);
    document.sb_event = document.getElementById("event");
    document.sb_board = document.getElementById("board");
    document.sb_effect = document.getElementById("effect");
    document.sb_message = document.getElementById("console-msg");
    document.sb_event_data = document.getElementById("event-data");
    document.sb_event_title = document.getElementById("event-title");
    setInterval(scroll_elements, 200);
    debug("Opening websocket..");
    let s = window.location.host + "/w";
    if (document.location.protocol.indexOf("https") >= 0) {
        s = "wss://" + s;
    } else {
        s = "ws://" + s;
    }
    document.sb_socket = new WebSocket(s);
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
function exit_game() {
    alert(messages[Math.floor(Math.random() * messages.length)]);
    return false;
}
function hamburger() {
    if (!is_mobile()) {
        return false;
    }
    let display_tab = document.getElementById("game-tab");
    if (display_tab === null) {
        return false;
    }
    if (display_tab.style.display.indexOf("none") > -1 || display_tab.style.display.length === 0) {
        display_tab.style.display = "block";
    } else {
        display_tab.style.display = "none";
    }
    return false;
}
function event_close() {
    document.sb_event.style.display = "none";
    document.sb_event_data.innerHTML = "";
    document.sb_event_title.innerText = "";
}
function auto_scroll() {
    if (!document.sb_auto) {
        return;
    }
    debug("Started auto scroll..");
    let tabs = document.getElementById("game-tab").children;
    let tabs_auto = false;
    for (let i = 0; i < tabs.length; i++) {
        if (tabs[i].id === "auto-tab") {
            continue;
        }
        if (tabs[i].id === "game-tweet-tab") {
            let tab_tweets = document.getElementById("game-tweet");
            if (tab_tweets === null || tab_tweets.children.length === 0) {
                continue;
            }
        }
        if (tabs_auto) {
            auto_set(tabs[i], tabs);
            return
        }
        tabs_auto = tabs[i].classList.contains("auto-selected");
        if (tabs_auto) {
            tabs[i].classList.remove("auto-selected");
        }
    }
    if (tabs[0].id !== "auto-tab") {
        auto_set(tabs[0], tabs);
    } else if (tabs[1].id !== "auto-tab") {
        auto_set(tabs[1], tabs);
    }
}
function recv(message) {
    if (message.data === null && !document.sb_loaded) {
        document.sb_socket.close();
        return;
    }
    if (!document.sb_loaded) {
        let load_message = document.getElementById("game-status-load");
        if (load_message !== null) {
            load_message.remove();
        }
    }
    update_board(message.data);
    if (!document.sb_loaded) {
        if (is_mobile()) {
            navigate("overview")
        } else {
            navigate("auto");
            check_mobile();
        }
        let game_status = document.getElementById("game-status-name");
        if (game_status !== null) {
            game_status.setAttribute("onclick", "return navigate('overview');");
        }
        document.sb_loaded = true;
    }
}
function update_tabs() {
    let tabs = document.getElementById("game-tab");
    if (tabs === null || tabs.length <= 3) {
        return;
    }
    for (let i = 0; i < tabs.children.length; i++) {
        if (tabs.children[i].id.indexOf("-team-") > 0) {
            let tabs_titles = document.getElementById(tabs.children[i].id.replace("-tab", ""));
            if (tabs_titles === null) {
                tabs.children[i].remove();
            }
        }
    }
    let teams = document.getElementsByClassName("team");
    for (let i = 0; i < teams.length; i++) {
        let team_tab = document.getElementById(teams[i].id + "-tab");
        if (teams[i].classList.contains("mini")) {
            if (team_tab !== null) {
                team_tab.remove();
            }
            continue;
        }
        if (team_tab === null) {
            team_tab = document.createElement("a");
            team_tab.id = teams[i].id + "-tab";
            let game_tab = document.getElementById("game-tab");
            if (game_tab !== null) {
                game_tab.appendChild(team_tab);
            }
        }
        let team_name = document.getElementById(teams[i].id + "-name-name");
        if (team_name === null) {
            continue;
        }
        if (team_name.scrollHeight > team_name.offsetHeight) {
            team_name.classList.add("small");
        }
        team_tab.setAttribute("href", "#");
        team_tab.setAttribute("onclick", "return navigate('" + teams[i].id + "');");
        let team_div = document.getElementById(teams[i].id);
        if (team_div !== null) {
            team_div.setAttribute("onclick", "return navigate('" + teams[i].id + "');");
        }
        team_tab.innerText = team_name.innerText
    }
}
function check_mobile() {
    if (is_mobile(true)) {
        return;
    }
    let menu_bar = document.getElementById("menu");
    let tabs = document.getElementById("game-tab");
    if (tabs === null || menu_bar === null) {
        return;
    }
    if (tabs.offsetHeight <= 25) {
        if (document.sb_tab_offset === null || (document.sb_tab_offset !== null && document.sb_tab_offset < tabs.parentElement.offsetWidth)) {
            document.sb_tab_offset = null;
            tabs.classList.remove("mobile");
            menu_bar.classList.remove("mobile");
            if (tabs.style.display.indexOf("none") > -1 || tabs.style.display.length === 0) {
                tabs.style.display = "";
            }
            return;
        } else if (document.sb_tab_offset === null) {
            return;
        }
    }
    document.sb_tab_offset = tabs.offsetWidth;
    tabs.classList.add("mobile");
    menu_bar.classList.add("mobile");
}
function callout_done() {
    document.sb_callout = false
    setTimeout(callout_hide, 1250);
}
function debug(message) {
    if (document.sb_debug) {
        console.log("DEBUG: " + message);
    }
}
function navigate(panel) {
    let overview = document.getElementById("overview-tab");
    let teams = document.getElementById("game-tab").children;
    select_div(panel, teams, true);
    if (panel === "auto") {
        if (overview !== null) {
            document.sb_auto = true;
            overview.classList.add("auto-selected");
            setTimeout(auto_scroll, interval_all);
        }
    } else if (document.sb_auto) {
        let overview_tab = document.getElementById("overview-tab");
        if (overview_tab !== null) {
            overview_tab.classList.remove("auto-selected");
        }
        document.sb_auto = false;
    }
    if (is_mobile()) {
        let game_tab = document.getElementById("game-tab");
        if (game_tab !== null) {
            if (game_tab.style.display.indexOf("none") === -1 || game_tab.style.display.length !== 0) {
                game_tab.style.display = "none";
            }
        }
    }
}
function display_close() {
    debug("Displaying closed board...");
    let disconnect_message = document.getElementById("game-disconnected");
    if (disconnect_message !== null) {
        disconnect_message.style.display = "block";
    }
}
function update_beacons() {
    let beacons = document.getElementsByClassName("beacon");
    for (let i = 0; i < beacons.length; i++) {
        set_beacon_image(beacons[i]);
    }
}
function display_invalid() {
    debug("Displaying invalid board...");
    if (!document.sb_loaded) {
        let load_message = document.getElementById("game-status-load");
        if (load_message !== null) {
            load_message.remove();
        }
    }
    let error_message = document.getElementById("game-invalid");
    if (error_message !== null) {
        error_message.style.display = "block";
    }
}
function scroll_elements() {
    let beacons = document.getElementsByClassName("team-beacon");
    for (let i = 0; i < beacons.length; i++) {
        scroll_beacon(beacons[i]);
    }
    let teams = document.getElementsByClassName("team-name-div");
    for (let i = 0; i < teams.length; i++) {
        // The simple scroll is broke here, so I made this!
        scroll_vert_simple(teams[i]);
    }
    /* Need to figure out why Host names do not scroll, even when set as 'block'.
    var hn = document.getElementsByClassName("host-name");
    for (var hi = 0; hi < hn.length; hi++) {
        scroll_element(hn[hi]);
    }
    */
}
function update_board(data) {
    let updates = JSON.parse(data);
    debug("Received " + updates.length + " entries...");
    for (let i = 0; i < updates.length; i++) {
        handle_update(updates[i]);
    }
    update_tabs();
    update_beacons();
    let game_name = document.getElementById("game-status-name");
    if (game_name !== null) {
        document.title = game_name.innerText;
    }
    callout_add("host", "callout-host");
    callout_add("service", "callout-service");
    callout_add("score-health", "callout-health");
    callout_add("score-flag-open", "callout-flag-open");
    callout_add("score-flag-lost", "callout-flag-lost");
    callout_add("score-ticket-open", "callout-ticket-open");
    callout_add("team-beacon-container", "callout-beacons");
    callout_add("score-ticket-closed", "callout-ticket-closed");
    callout_add("score-flag-captured", "callout-flag-captured");
}
function scroll_element(ele) {
    if (ele.scrollWidth === 0) {
        ele.classList.remove("reverse");
        return;
    }
    let reversed = ele.classList.contains("reverse");
    if ((ele.scrollWidth - ele.offsetWidth) === ele.scrollLeft) {
        ele.classList.add("reverse");
        reversed = true;
    } else if (ele.scrollLeft === 0) {
        ele.classList.remove("reverse");
        reversed = false;
    }
    if (reversed) {
        ele.scrollLeft -= 2;
    } else {
        ele.scrollLeft += 2;
    }
}
function handle_event(event) {
    if (!event.value) {
        return;
    }
    if (event.value === "1" || event.value === "3") {
        handle_event_popup(event);
        return;
    }
    if (event.value === "0") {
        handle_event_message(event)
        return;
    }
    if (event.value === "2") {
        handle_event_effect(event)
        return;
    }
}
function callout(event, type) {
    if (is_mobile()) {
        return;
    }
    if (event.fromElement === null) {
        return;
    }
    if (type === "callout-host" && !event.fromElement.classList.contains("offline")) {
        return;
    }
    let callout = document.getElementById("callout");
    if (callout === null) {
        return;
    }
    callout.style.display = "block";
    callout.style.top = (event.clientY + 10) + "px";
    callout.style.left = (event.clientX + 10) + "px";
    for (let i = 0; i < callout.children.length; i++) {
        if (callout.children[i].id === type) {
            callout.children[i].style.display = "block";
        } else {
            callout.children[i].style.display = "none";
        }
    }
    document.sb_callout = true;
}
function handle_update(update) {
    debug("Processing '" + JSON.stringify(update) + "'..")
    if (update.event) {
        handle_event(update);
        return
    }
    if (update.remove) {
        let removals = document.getElementById(update.id);
        if (removals !== null) {
            debug("Deleting ID " + update.id + "..");
            removals.remove();
            return
        }
    }
    let parent = null;
    let seperator = update.id.lastIndexOf("-");
    if (seperator > 0) {
        parent = document.getElementById(update.id.substring(0, seperator));
    }
    let target = document.getElementById(update.id);
    if (target === null && parent !== null) {
        target = document.createElement("div")
        target.id = update.id;
        parent.appendChild(target);
        debug("Created element " + update.id + "..");
    }
    if (target === null) {
        return;
    }
    if (update.class) {
        target.className = "";
        let classes = update.class.split(" ");
        for (let i = 0; i < classes.length; i++) {
            target.classList.add(classes[i]);
        }
    }
    if (!update.value) {
        return;
    }
    if (!update.name) {
        target.innerText = update.value;
        return;
    }
    if (update.name !== "class") {
        target.style[update.name] = update.value;
        return;
    }
    let class_action = update.value[0];
    if (!(class_action === "-" || class_action === "+")) {
        target.classList.add(update.value);
        return;
    }
    let class_data = update.value.substring(1);
    if (class_action === "+") {
        if (!target.classList.contains(class_data)) {
            target.classList.add(class_data);
        }
    }
    if (class_action === "-") {
        if (target.classList.contains(class_data)) {
            target.classList.remove(class_data);
        }
    }
}
function scroll_beacon(beacon) {
    let bc = beacon.children[0];
    if (bc === null || bc.length === 0) {
        return;
    }
    if (bc.offsetHeight <= beacon.offsetHeight) {
        beacon.classList.remove("reverse");
        return;
    }
    let rev = beacon.classList.contains("reverse");
    if ((beacon.scrollTop + beacon.offsetHeight) === bc.offsetHeight) {
        beacon.classList.add("reverse");
        rev = true;
    } else if (beacon.scrollTop === 0) {
        beacon.classList.remove("reverse");
        rev = false;
    }
    if (rev) {
        beacon.scrollTop -= 5;
    } else {
        beacon.scrollTop += 5;
    }
}
function callout_add(ele, type) {
    let elements = document.getElementsByClassName(ele);
    for (let i = 0; i < elements.length; i++) {
        if (!elements[i].onmouseover) {
            elements[i].setAttribute("onmouseover", "callout(event, '" + type + "');");
            elements[i].setAttribute("onmouseout", "callout_done();");
        }
    }
}
function auto_set(div, entries) {
    div.classList.add("auto-selected");
    if (div.id === "overview-tab") {
        setTimeout(auto_scroll, interval_all);
    } else if (div.id === "credits-tab") {
        setTimeout(auto_scroll, interval_credit);
    } else {
        setTimeout(auto_scroll, interval_team);
    }
    callout_hide(true);
    select_div(div.id.replace("-tab", ""), entries, false);
}
function scroll_vert_simple(ele) {
    if (ele.scrollWidth === 0) {
        ele.classList.remove("reverse");
        return;
    }
    let rev = ele.classList.contains("reverse");
    if (!rev && ele.scrollTop === 0) {
        ele.scrollTop = ele.offsetHeight;
        ele.classList.add("reverse");
        return;
    }
    if (rev && ele.scrollTop === 0) {
        ele.classList.remove("reverse");
        return;
    }
    ele.scrollTop -= 2;
}
function set_beacon_image(beacon) {
    if (beacon.style.background.indexOf("data") >= 0) {
        return
    }
    let bg_color = beacon.style.background;
    if (bg_color === null || bg_color === "") {
        return;
    }
    let color = [0, 0, 0];
    if (bg_color.indexOf(",") > 0) {
        let colors = bg_color.split(",");
        if (colors.length === 3) {
            color[0] = parseInt(colors[0].replace(")", "").replace("rgb(", ""), 10);
            color[1] = parseInt(colors[1].replace(")", "").replace("rgb(", ""), 10);
            color[2] = parseInt(colors[2].replace(")", "").replace("rgb(", ""), 10);
        }
    } else {
        let colors = bg_color.match(/[a-f0-9]{2}/gi);
        if (colors.length === 3) {
            color[0] = parseInt(colors[0], 16);
            color[1] = parseInt(colors[1], 16);
            color[2] = parseInt(colors[2], 16);
        }
    }
    let image = new Image();
    image.onload = function () {
        let canvas = document.createElement("canvas");
        canvas.width = 25;
        canvas.height = 25;
        let layer = canvas.getContext("2d");
        layer.drawImage(this, 0, 0);
        let draw = layer.getImageData(0, 0, 128, 128);
        for (let i = 0; i < draw.data.length; i += 4) {
            draw.data[i] = color[0];
            draw.data[i + 1] = color[1];
            draw.data[i + 2] = color[2];
        }
        layer.putImageData(draw, 0, 0);
        beacon.style.background = "url('" + canvas.toDataURL("image/png") + "')";
    }
    image.crossOrigin = "anonymous";
    image.src = "/image/beacon.png";
}
function handle_event_popup(event) {
    if (event.remove) {
        document.sb_event.style.display = "none";
        document.sb_event_data.innerHTML = "";
        document.sb_event_title.innerText = "";
        debug("Removed window event.");
        return;
    }
    if (event.data.title) {
        document.sb_event_title.innerText = event.data.title;
    } else {
        document.sb_event_title.innerText = "COMPROMISE DETECTED!!!";
    }
    if (!event.data.fullscreen || event.data.fullscreen.toLowerCase() === "false") {
        document.sb_event.classList.remove("fullscreen");
    } else {
        document.sb_event.classList.add("fullscreen");
    }
    if (event.value === "3") {
        if (!(event.data.video)) {
            return;
        }
        let youtube_url = "https://www.youtube-nocookie.com/embed/" + event.data.video + "?controls=0&amp;autoplay=1";
        if (event.data.start) {
            youtube_url = youtube_url + "&amp;start=" + event.data.start;
        }
        document.sb_event_data.innerHTML = '<iframe width="100%" height=100%" src="' + youtube_url + '" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope"></iframe>';
        document.sb_event.style.display = "block";
        debug("Video event shown!");
    } else {
        document.sb_event_data.innerHTML = "<div>" + event.data.text + "</div>";
        document.sb_event.style.display = "block";
        debug("Window event shown!");
    }
}
function handle_event_effect(event) {
    if (event.remove) {
        let effect = document.getElementById("eff-" + event.id);
        if (effect !== null) {
            effect.remove();
        }
        return;
    }
    let container = document.createElement("div");
    container.id = "eff-" + event.id;
    container.innerHTML = event.data.html;
    document.sb_effect.appendChild(container);
    debug("Added effect event!");
    let scripts = container.getElementsByTagName("script");
    if (scripts.length === 0) {
        return;
    }
    debug("Triggering event scripts");
    for (let i = 0; i < scripts.length; i++) {
        eval(scripts[i].text);
    }
}
function handle_event_message(event) {
    if (event.remove) {
        let event_message = document.getElementById("msg-" + event.id);
        if (event_message !== null) {
            event_message.remove();
        }
        return;
    }
    let message = document.createElement("div");
    message.id = "msg-" + event.id;
    message.classList.add("message");
    if (event.data.command && event.data.command.length > 0) {
        if (event.data.response && event.data.response.length > 0) {
            message.innerHTML = "[root@localhost ~]# " + event.data.text + "<br/>" + event.data.response.replace("\n", "<br/>");
        } else {
            message.innerText = "[root@localhost ~]# " + event.data.text;
        }
    } else {
        message.innerText = "[root@localhost ~]# echo " + event.data.text + " > /dev/null";
    }
    document.sb_message.appendChild(message);
    document.sb_message.scrollTop = document.sb_message.offsetHeight;
    debug("Added message event!");
}
function is_mobile(css_only = false) {
    let media_match = window.matchMedia("only screen and (max-width: 650px)").matches || window.matchMedia("only screen and (max-width:767px) and (orientation:portrait)").matches
    if (media_match) {
        return true;
    }
    if (css_only) {
        return false;
    }
    let menu = document.getElementById("menu");
    if (menu === null || !menu) {
        return false;
    }
    return menu.classList.contains("mobile");
}
function callout_hide(ignore = false) {
    if (!document.sb_callout || ignore) {
        let callout = document.getElementById("callout");
        if (callout === null) {
            return;
        }
        callout.style.display = "none";
    }
}
function select_div(panel, entries, select) {
    for (let i = 0; i < entries.length; i++) {
        let name = entries[i].id.replace("-tab", "");
        if (select) {
            if (name === panel) {
                entries[i].classList.add("selected");
            } else {
                entries[i].classList.remove("selected");
                entries[i].classList.remove("auto-selected");
            }
        }
        let target = document.getElementById(name);
        if (target !== null) {
            if (name === panel) {
                target.classList.add("selected");
            } else {
                target.classList.remove("selected");
            }
        }
    }
    let team = document.getElementById("game-team");
    if (team === null) {
        return;
    }
    if (panel === "auto" || panel === "overview") {
        team.classList.remove("single");
    } else {
        team.classList.add("single");
    }
}
