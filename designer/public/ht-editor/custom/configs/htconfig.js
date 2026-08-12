window.htconfig = {
    Default: {
        toolTipDelay: 100,
        toolTipContinual: true,
		mockTouch: true,
        autoHideScrollBar: window.navigator.platform.indexOf('Win') > -1 || !("ontouchend" in document),
        convertURL: function (url) {
            var storagePrefix = '';
            if (storagePrefix && url && !/^data:image/.test(url) && !/^http/.test(url) && !/^https/.test(url)) {
                url = storagePrefix + '/' + url
            }
            // 获取工程信息 add by lei.zhang
            var searchArr = window.location.search ? window.location.search.split('=') : [''];
            var projectInfo = searchArr[searchArr.length - 1];
            if (window.location.href.indexOf('kingclient3d') >= 0) {
                // kf3.6
                url = `/kingclient3d/${projectInfo}/${url}`;
            } else {
                // kp2.0
                url = `/${projectInfo}/${url}`;
            }
            // append timestamp
            // append sid
            var match = window.location.href.match('sid=([0-9a-z\-]*)');
            if (match) {
                window.sid = match[1]
            }
            if (window.sid) {
                url += '&sid=' + window.sid;
            }
            return url;
        }
    }
};