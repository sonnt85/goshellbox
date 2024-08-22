/**
 * @thanhson.rf@gmail.com
 */

// init
(function (W) {
    W.DataType = {
        Err: 0,
        Data: 1,
        Resize: 2,
        KeepConnect: 3,
    };
    W.FitAddon = FitAddon.FitAddon;
    W.WebglAddon = WebglAddon.WebglAddon;
    W.WebLinksAddon = WebLinksAddon.WebLinksAddon;

    W.NewWebSocket = function (path) {
        return new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + location.pathname + "cmd/" + path);
    };
    W.NewTerminal = function () {
        // rightClickSelectsAll: true
        return new Terminal({ useStyle: true, screenKeys: true });
    };
    W.GetByAjax = function (url, callback) {
        var ajax = new XMLHttpRequest();
        ajax.open("GET", url);
        ajax.onreadystatechange = function () {
            if (ajax.readyState == 4 && ajax.status == 200) {
                callback(JSON.parse(ajax.responseText), ajax);
            }
        }
        ajax.send();
    }
})(window);

// goshellbox login:
// Password:
// Login incorrect

// goshellbox component
window.ShellBox = function (dom) {
    var conn = null;
    var term = NewTerminal();
    var fitAddon = new FitAddon();
    var webglAddon = new WebglAddon();
    var webLinksAddon = new WebLinksAddon();
    var sendData = function (dataType, data) {
        if (conn != null) {
            conn.send(JSON.stringify({ 't': dataType, "d": data }));
        }
    };

    var onLoginSuccess = function (path) {
        conn = NewWebSocket(path);
        // websocket connect
        conn.onclose = function (e) {
            term.writeln("connection closed.");
        };

        var lastTime = new Date().getTime();
        window.onmousemove = function () {
            lastTime = new Date().getTime();
        };

        setInterval(function () {
            sendData(DataType.KeepConnect, "");
            if (new Date().getTime() - lastTime > 3600000) {
                conn.close();
            }
        }, 30000);
        conn.onopen = function () {
            fitAddon.fit();
            sendData(DataType.Resize, [term.cols, term.rows]);
        };

        conn.onmessage = function (event) {
            var lastTime = new Date().getTime();

            // window.data = event.data;
            var data = event.data;
            if (data instanceof Blob) {
                var fileReader = new FileReader();
                fileReader.onload = function (e) {
                    var textData = e.target.result;
                    if (fileReader.error) {
                        console.error(fileReader.error);
                    } else {
                        term.write(textData);
                    }
                };
                fileReader.readAsText(data);
            } else if (typeof data === 'string') {
                term.write(data);
            } else {
                var hexStr = '';
                for (var i = 0; i < data.length; i++) {
                    hexStr += data.charCodeAt(i).toString(16).padStart(2, '0') + ' ';
                }
                term.write("00000000  " + hexStr + "\n");
            }
        };

        term.onResize(function (data) {
            sendData(DataType.Resize, [data.cols, data.rows]);
        });
        term.onData(function (data) {
            sendData(DataType.Data, data);
        });
    };

    // terminal term
    term.onTitleChange(function (title) { document.title = title; });
    term.open(dom);

    // shellbox login module
    (function () {
        var isInput = true;
        var tag = 1;
        var username = "";
        var password = "";

        (function () {
            isInput = false;
            var token = ('goshellbox-token' in sessionStorage) ? sessionStorage.getItem("goshellbox-token") : '';
            GetByAjax("login?token=" + token, function (data) {
                if (data.code == 0) {
                    isInput = false;
                    onLoginSuccess(data.path);
                } else {
                    isInput = true;
                    term.write("ShellBox login:");
                }
            });
        })();

        var doLogin = function () {
            isInput = false;
            GetByAjax("login?username=" + username + "&password=" + password, function (data) {
                tag = 1;
                username = "";
                password = "";
                if (data.code == 0) {
                    isInput = false;
                    sessionStorage.setItem("goshellbox-token", data.path);
                    onLoginSuccess(data.path);
                } else {
                    isInput = true;
                    term.writeln(data.msg);
                    term.write("\nShellBox login:");
                }
            });
        }
        var BackSpacePrev = [27, 91, 63, 50, 53, 108, 13].map(function (e) { return String.fromCharCode(e) }).join('');
        var BackSpaceNext = [27, 91, 48, 75, 27, 91, 63, 50, 53, 104].map(function (e) { return String.fromCharCode(e) }).join('');

        term.onData(function (data) {
            if (!isInput) return;
            if (data.charCodeAt(0) == 13) {
                term.writeln("");
                if (tag == 1) {
                    tag++;
                    term.write("Password:");
                } else {
                    if (username.substr(-1) !== '.') {
                        username = md5(username);
                        password = md5(password);
                    } else {
                        username = encodeURI(username);
                        password = encodeURI(password);
                    }
                    doLogin();
                }
            } else if (data.charCodeAt(0) == 127) {
                if (tag == 1) {
                    username = username.substr(0, username.length - 1);
                    term.write(BackSpacePrev + "ShellBox login:" + username + BackSpaceNext);
                } else {
                    password = password.substr(0, password.length - 1);
                }
            } else {
                if (tag == 1) {
                    term.write(data);
                    username += data;
                } else {
                    password += data;
                }
            }

        });
    })();


    term.loadAddon(fitAddon);
    // if (typeof WebGLRenderingContext !== "undefined") {
    //     var canvas = document.createElement("canvas");
    //     var gl = canvas.getContext("webgl") || canvas.getContext("experimental-webgl");
    //     if (gl && gl instanceof WebGLRenderingContext) {
    term.loadAddon(webglAddon);
    //     }
    // }
    term.loadAddon(webLinksAddon);
    // logoutButton
    // Logout handler in the ShellBox function  
    var onLogout = function () {
        term.write('\x03'); // Send ctrl+c to kill any running process
        conn.close(); // Close the WebSocket connection to the server
        // Make a request to the logout endpoint defined in your Go code.  

        var token = sessionStorage.getItem("goshellbox-token"); // Assuming you have a function to retrieve the token from local storage
        if (!token) {
            term.writeln("Not logged in yet");
            return;
        }
        sessionStorage.removeItem("goshellbox-token");
        term.writeln("Logging out...");
        setTimeout(function () {
            location.reload();
        }, 1000);
    };

    var logoutButton = document.createElement('button');
    logoutButton.textContent = 'Logout';
    logoutButton.style.position = 'fixed';
    logoutButton.style.bottom = '10px';
    logoutButton.style.right = '10px';
    logoutButton.style.backgroundColor = '#f5222d';
    logoutButton.style.color = 'white';
    logoutButton.style.borderRadius = '3px';
    logoutButton.style.padding = '5px 15px';
    logoutButton.style.cursor = 'pointer';
    logoutButton.style.fontWeight = 'bold';
    logoutButton.style.zIndex = 100; // Ensure the button is displayed above the terminal  

    document.body.appendChild(logoutButton); // Add the button to the body  


    // Event listener for the logout button  
    logoutButton.addEventListener('click', onLogout);

    this.fit = function () {
        fitAddon.fit();
    };

    this.term = term;
    this.conn = conn;
    dom.oncontextmenu = function (event) {
        return true;
        if (term.hasSelection()) {
            event.preventDefault();
            return false;
        }
        return true;
    }
};

// run
window.onload = function () {
    var dom = document.createElement("div");
    dom.className = "console";
    document.body.appendChild(dom);
    this.singleShellBox = new ShellBox(dom);
    this.onresize = function () {
        this.singleShellBox.fit();
    }
    this.onresize();
};
