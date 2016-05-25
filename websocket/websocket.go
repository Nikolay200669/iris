package websocket

import (
	"github.com/kataras/iris/config"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/logger"
)

// to avoid the import cycle to /kataras/iris, because at the future I am thinking of bundle the ws package
// inside Iris' configuration like kataras/iris/sessions, kataras/iris/render/rest, kataras/iris/render/template, kataras/iris/server and so on.
type irisStation interface {
	H_(string, string, func(context.IContext))
	StaticContent(string, string, []byte)
	Logger() *logger.Logger
}

//

// New returns a new running websocket server
func New(station irisStation, path string) Server {
	server := newServer(config.DefaultAutoStart)

	station.H_("GET", path, func(ctx context.IContext) {
		if err := server.Upgrade(ctx); err != nil {
			station.Logger().Panic(err)
		}
	})

	// serve the client side on domain:port/iris-ws.js
	station.StaticContent("/iris-ws.js", "application/json", clientSource)

	return server
}

var clientSource = []byte(`var stringMessageType = 0;
var intMessageType = 1;
var boolMessageType = 2;
// bytes is missing here for reasons I will explain somewhen
var jsonMessageType = 4;
var prefix = "iris-websocket-message:";
var separator = ";";
var prefixLen = prefix.length;
var separatorLen = separator.length;
var prefixAndSepIdx = prefixLen + separatorLen - 1;
var prefixIdx = prefixLen - 1;
var separatorIdx = separatorLen - 1;
var Ws = (function () {
    //
    function Ws(endpoint, protocols) {
        var _this = this;
        // events listeners
        this.connectListeners = [];
        this.disconnectListeners = [];
        this.nativeMessageListeners = [];
        this.messageListeners = {};
        if (!window["WebSocket"]) {
            return;
        }
        if (endpoint.indexOf("ws") == -1) {
            endpoint = "ws://" + endpoint;
        }
        if (protocols != null && protocols.length > 0) {
            this.conn = new WebSocket(endpoint, protocols);
        }
        else {
            this.conn = new WebSocket(endpoint);
        }
        this.conn.onopen = (function (evt) {
            _this.fireConnect();
            _this.isReady = true;
            return null;
        });
        this.conn.onclose = (function (evt) {
            _this.fireDisconnect();
            return null;
        });
        this.conn.onmessage = (function (evt) {
            _this.messageReceivedFromConn(evt);
        });
    }
    //utils
    Ws.prototype.isNumber = function (obj) {
        return !isNaN(obj - 0) && obj !== null && obj !== "" && obj !== false;
    };
    Ws.prototype.isString = function (obj) {
        return Object.prototype.toString.call(obj) == "[object String]";
    };
    Ws.prototype.isBoolean = function (obj) {
        return typeof obj === 'boolean' ||
            (typeof obj === 'object' && typeof obj.valueOf() === 'boolean');
    };
    Ws.prototype.isJSON = function (obj) {
        try {
            JSON.parse(obj);
        }
        catch (e) {
            return false;
        }
        return true;
    };
    //
    // messages
    Ws.prototype._msg = function (event, messageType, dataMessage) {
        return prefix + event + separator + String(messageType) + separator + dataMessage;
    };
    Ws.prototype.encodeMessage = function (event, data) {
        var m = "";
        var t = 0;
        if (this.isNumber(data)) {
            t = intMessageType;
            m = data.toString();
        }
        else if (this.isBoolean(data)) {
            t = boolMessageType;
            m = data.toString();
        }
        else if (this.isString(data)) {
            t = stringMessageType;
            m = data.toString();
        }
        else if (this.isJSON(data)) {
            //propably json-object
            t = jsonMessageType;
            m = JSON.stringify(data);
        }
        else {
            console.log("Invalid");
        }
        return this._msg(event, t, m);
    };
    Ws.prototype.decodeMessage = function (event, websocketMessage) {
        //iris-websocket-message;user;4;themarshaledstringfromajsonstruct
        var skipLen = prefixLen + separatorLen + event.length + 2;
        if (websocketMessage.length < skipLen + 1) {
            return null;
        }
        var messageType = parseInt(websocketMessage.charAt(skipLen - 2));
        var theMessage = websocketMessage.substring(skipLen, websocketMessage.length);
        if (messageType == intMessageType) {
            return parseInt(theMessage);
        }
        else if (messageType == boolMessageType) {
            return Boolean(theMessage);
        }
        else if (messageType == stringMessageType) {
            return theMessage;
        }
        else if (messageType == jsonMessageType) {
            return JSON.parse(theMessage);
        }
        else {
            return null; // invalid
        }
    };
    Ws.prototype.getCustomEvent = function (websocketMessage) {
        if (websocketMessage.length < prefixAndSepIdx) {
            return "";
        }
        var s = websocketMessage.substring(prefixAndSepIdx, websocketMessage.length);
        var evt = s.substring(0, s.indexOf(separator));
        return evt;
    };
    Ws.prototype.getCustomMessage = function (event, websocketMessage) {
        var eventIdx = websocketMessage.indexOf(event + separator);
        var s = websocketMessage.substring(eventIdx + event.length + separator.length + 2, websocketMessage.length);
        return s;
    };
    //
    // Ws Events
    // messageReceivedFromConn this is the func which decides
    // if it's a native websocket message or a custom iris-ws message
    // if native message then calls the fireNativeMessage
    // else calls the fireMessage
    //
    // remember Iris gives you the freedom of native websocket messages if you don't want to use this client side at all.
    Ws.prototype.messageReceivedFromConn = function (evt) {
        //check if iris-ws message
        var message = evt.data;
        if (message.indexOf(prefix) != -1) {
            var event_1 = this.getCustomEvent(message);
            if (event_1 != "") {
                // it's a custom message
                this.fireMessage(event_1, this.getCustomMessage(event_1, message));
                return;
            }
        }
        // it's a native websocket message
        this.fireNativeMessage(message);
    };
    Ws.prototype.OnConnect = function (fn) {
        if (this.isReady) {
            fn();
        }
        this.connectListeners.push(fn);
    };
    Ws.prototype.fireConnect = function () {
        for (var i = 0; i < this.connectListeners.length; i++) {
            this.connectListeners[i]();
        }
    };
    Ws.prototype.OnDisconnect = function (fn) {
        this.disconnectListeners.push(fn);
    };
    Ws.prototype.fireDisconnect = function () {
        for (var i = 0; i < this.disconnectListeners.length; i++) {
            this.disconnectListeners[i]();
        }
    };
    Ws.prototype.OnMessage = function (cb) {
        this.nativeMessageListeners.push(cb);
    };
    Ws.prototype.fireNativeMessage = function (websocketMessage) {
        for (var i = 0; i < this.nativeMessageListeners.length; i++) {
            this.nativeMessageListeners[i](websocketMessage);
        }
    };
    Ws.prototype.On = function (event, cb) {
        if (this.messageListeners[event] == null || this.messageListeners[event] == undefined) {
            this.messageListeners[event] = [];
        }
        this.messageListeners[event].push(cb);
    };
    Ws.prototype.fireMessage = function (event, message) {
        for (var key in this.messageListeners) {
            if (this.messageListeners.hasOwnProperty(key)) {
                if (key == event) {
                    for (var i = 0; i < this.messageListeners[key].length; i++) {
                        this.messageListeners[key][i](message);
                    }
                }
            }
        }
    };
    //
    // Ws Actions
    Ws.prototype.Disconnect = function () {
        this.conn.close();
    };
    // EmitMessage sends a native websocket message
    Ws.prototype.EmitMessage = function (websocketMessage) {
        this.conn.send(websocketMessage);
    };
    // Emit sends an iris-custom websocket message
    Ws.prototype.Emit = function (event, data) {
        var messageStr = this.encodeMessage(event, data);
        this.EmitMessage(messageStr);
    };
    return Ws;
}());
`)
