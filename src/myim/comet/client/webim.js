(function(win) {
    // option:
    //    function onMessage(msg) : receive a message
    //    function onStart: login success
    //    function onStop: disconnect
    var WebIm = function(wsUrl, appId, userId, token, targetType, targetId, options) {
        this.wsUrl = wsUrl;
        this.appId = appId;
        this.userId = userId;
        this.token = token;
        this.options = options || {};

        this.connected = false;
        this.auth = false;

        this.targetType = targetType;
        this.targetId = targetId;

        this.seqId = 1;
        this.lastMsgId = 0;
        this.lastSyncMs = 0;

        this.msgReplyCb = new Map();
        this.msgSyncCb = new Map();

        this.ws = null;
        this.createConnect();
    }

    WebIm.VER = 1;

	WebIm.RC_OK = 0;
    WebIm.RC_ERROR = 1;

    WebIm.CHAT_TYPE_SINGLE = 0;
    WebIm.CHAT_TYPE_GROUP = 1;

    WebIm.DIRECTION_FORWARD = 1;
    WebIm.DIRECTION_BACKWARD = 2;

	// login
	WebIm.OP_LOGIN = 0;
	WebIm.OP_LOGIN_REPLY = 1;

	// heartbeat
	WebIm.OP_HEARTBEAT       = 2;
	WebIm.OP_HEARTBEAT_REPLY = 3;

	// send messgae
	WebIm.OP_SEND_MSG       = 4;
	WebIm.OP_SEND_MSG_REPLY = 5;

	// push msg notify
	WebIm.OP_MSG_NOTIFY     = 6;
	WebIm.OP_MSG_NOTIFY_ACK = 7;

	// msg sync
	WebIm.OP_MSG_SYNC       = 8;
	WebIm.OP_MSG_SYNC_REPLY = 9;

	// logout
	WebIm.OP_LOGOUT       = 10;
    WebIm.OP_LOGOUT_REPLY = 11;

    WebIm.prototype.createConnect = function() {
        var self = this;

        if (self.connected) {
            console.log("already connected")
            return;
        }

        connect();

        var heartbeatInterval;
        var msgSyncInterval;
        function connect() {
            var ws = new WebSocket(self.wsUrl);
            self.ws = ws;

            ws.onopen = function() {
                self.connected = true;
                auth();
            }

            ws.onmessage = function(evt) {
                var data = evt.data;

                console.log("receive msg data:" + data);

                var proto;
                try {
                    proto = JSON.parse(data);
                } catch(e) {
                    console.log("parse msg data failed.err:" + e)
                    return;
                }
                body = proto.body;

                switch(proto.op) {
                    case WebIm.OP_LOGIN_REPLY:
                        // heartbeat
                        heartbeat();
                        heartbeatInterval = setInterval(heartbeat, body.heartbeat * 1000);

                        // msg sync
                        msgSync(true)
                        msgSyncInterval = setInterval(function(){msgSync(false)}, 3000);

                        notify = self.options.onStart;
                        self.auth = true;
                        if (notify) notify();

                        break;

                    case WebIm.OP_HEARTBEAT_REPLY:
                        // heartbeat reply
                        console.log("receive: heartbeat");
                        break;

                    case WebIm.OP_SEND_MSG_REPLY:
                        console.log("receive send msg reply");

                        var cb = null;
                        if (proto.seq > 0) {
                            cb = self.msgReplyCb.get(proto.seq);
                        }

                        if (body.retCode == WebIm.RC_OK) {

                            self.lastSyncMs = new Date().getTime();

                            if (self.lastMsgId == 0) { // the first msg
                                self.lastMsgId = body.msgId;
                            } else if (self.lastMsgId == body.preMsgId) { // no gap
                                self.lastMsgId = body.msgId;
                            } else if (body.preMsgList && self.options.onMessage) { // gap msg notify
                                console.log("found gap msgs.len:" + body.preMsgList.length)
                                notify = self.options.onMessage;
                                if (notify) {
                                    for (var i=0; i<body.preMsgList.length; i++) {
                                        if (body.preMsgList[i].preMsgId == self.lastMsgId) {
                                            self.lastMsgId = body.preMsgList[i].msgId;
                                            notify(body.preMsgList[i])
                                        }
                                    }
                                }
                                self.lastMsgId = body.msgId;
                            }
                            if (cb && cb.cbOk) cb.cbOk(body.msgId, body.preMsgId);
                        } else {
                            if (cb && cb.cbErr) cb.cbErr();
                        }

                        if (cb) self.msgReplyCb.delete(proto.seq);

                        break;

                    case WebIm.OP_MSG_NOTIFY:
                        console.log("receive msg notify")
                        var msg = body.msg;
                        if (msg) {
                            self.lastSyncMs = new Date().getTime();
                            notify = self.options.onMessage;
                            if (self.lastMsgId != 0 && msg.preMsgId != self.lastMsgId) {
                                console.log("found lost msgs, try to sync...")
                                msgSync(true);
                                return
                            }
                            lastMsgId = msg.msgId;
                            if (notify) notify(msg)
                        }
                        break;

                    case WebIm.OP_MSG_SYNC_REPLY:
                        console.log("receive sync msg reply");

                        var cb = null;
                        if (proto.seq > 0) {
                            cb = self.msgSyncCb.get(proto.seq)
                        }
                        if (body.retCode == WebIm.RC_OK) {
                            if (cb && cb.cbOk) cb.cbOk(body.msgList);
                        } else {
                            if (cb && cb.cbErr) cb.cbErr();
                        }
                        if (cb) self.msgSyncCb.delete(proto.seq);

                        break;
                }
            }

            ws.onclose = function() {
                console.log("disconnect")

                self.connected = false;
                self.auth = false;
                if (heartbeatInterval) clearInterval(heartbeatInterval);
                if (msgSyncInterval) clearInterval(msgSyncInterval);

                notify = self.options.onStop;
                if (notify) notify();
            }

            function heartbeat() {
                var proto = {
                    ver: WebIm.VER,
                    op: WebIm.OP_HEARTBEAT,
                    seq: self.seqId++,
                    body: null,
                };
                ws.send(JSON.stringify(proto));
                console.log("send: heartbeat");
            }

            function msgSync(force) {
                now = new Date().getTime();
                if (now - self.lastSyncMs > 60 * 1000 || force) {
                    self.lastSyncMs = now;
                    self.getHistoryMsgs({startMsgId: self.lastMsgId, direction: WebIm.DIRECTION_FORWARD},
                    function(msgList) {
                        if (msgList) {
                            notify = self.options.onMessage;
                            for (var i=0; i<msgList.length; i++) {
                                if (self.lastMsgId == 0 || msgList[i].preMsgId == self.lastMsgId) {
                                    self.lastMsgId = msgList[i].msgId;
                                    if (notify) notify(msgList[i])
                                }
                            }
                        }
                    }, null)
                }
            }

            function auth() {
                var proto = {
                    ver: WebIm.VER,
                    op: WebIm.OP_LOGIN,
                    seq: self.seqId++,
                    body: {
                        token: self.token,
                        userInfo: {
                            appId: self.appId, 
                            userId: self.userId,
                        }
                    },
                };
                ws.send(JSON.stringify(proto));
            }
        }
    }

    WebIm.prototype.reConnect = function() {
        var self = this;
        self.createConnect();
    }

    WebIm.prototype.logout = function() {
        var self = this;
        self.ws.close();
    }

    WebIm.prototype.getNextSeq = function() {
        var self = this;
        return self.seqId++;
    }

    WebIm.prototype.sendMsg = function(seq, msg, cbOk, cbErr) {
        var self = this;
        if (!self.auth) {
            console.log("not auth");
            return -1
        }
        if (seq < 0) {
            seq = self.seqId++;
        }

        // add msg reply listener
        self.msgReplyCb.set(seq, {cbOk: cbOk, cbErr: cbErr})

        var proto = {
            ver: WebIm.VER,
            op: WebIm.OP_SEND_MSG,
            seq: seq,
            body: {
                lastMsgId: self.lastMsgId,
                targetType: self.targetType, 
                targetId: self.targetId,
                msgData: msg,
            },
        };
        self.ws.send(JSON.stringify(proto));
        return seq;
    }

    WebIm.prototype.getHistoryMsgs = function(options, cbOk, cbErr) {
        var self = this;

        var seq = self.seqId++;

        // add msg sync listener
        self.msgSyncCb.set(seq, {cbOk: cbOk, cbErr: cbErr})

        var startMsgId = 0;
        var count = 10;
        var direction = WebIm.DIRECTION_BACKWARD;

        if (options && options.startMsgId > 0) {
            startMsgId = options.startMsgId;
        }
        if (options && options.direction) {
            direction = options.direction;
        }
        if (options && options.count) {
            count = options.count;
        }

        proto = {
            ver: WebIm.VER,
            op: WebIm.OP_MSG_SYNC,
            seq: seq,
            body: {
                targetType: self.targetType,
                targetId: self.targetId,
                direction: direction,
                startMsgId: startMsgId,
                count: count,
            },
        };
        self.ws.send(JSON.stringify(proto));
    }

    win['WebIm'] = WebIm;
})(window);
