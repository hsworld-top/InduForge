(function() {

    window.hteditor_config.handleEvent = function (editor, type, params) {
        if (type === 'SceneViewCreated') {

        }
        else if (type === 'SceneViewReloading') {

        }
        else if (type === 'SceneViewReloaded') {

        }
        else if (type === 'SceneViewOpening') {

        }
        else if (type === 'SceneViewOpened') {

        } else if (type === 'sceneSaved' && editor.isSaveInKP) {
            // 在KP界面上触发的保存,需要将编辑器界面关闭
            window.parent.postMessage('closed', editor3d.parentHref);
        }
    };

})();
























