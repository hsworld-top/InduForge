(function() {

    window.hteditor_config.handleEvent = function(editor, type, params) {
        var S = hteditor.getString;

        if (type === 'displayViewCreated' || type === 'displayViewOpened') {
            addSkewTranlateItem(params.displayView.graphView, editor);
            // addPrintSelectionItem(params.displayView.displayTree, 'editor.displayTree');
            // addPrintSelectionItem(params.displayView.graphView, 'editor.displayView.graphView');
        }
        else if (type === 'symbolViewCreated' || type === 'symbolViewOpened') {
            // addPrintSelectionItem(params.symbolView.symbolList, 'editor.symbolList');
            // addPrintSelectionItem(params.symbolView.graphView, 'editor.symbolView.graphView');
        }
        else if (type === 'displayViewSaving' || type === 'displayViewNewNameInputing') {
            if (type === 'displayViewSaving') {
                // 预览与快照均由平台生成，不允许场景 JSON 保留用户可控地址。
                params.displayView.dm.a('previewURL', undefined);
                params.displayView.dm.a('snapshotURL', undefined);
            }
            // if (!params.displayView.dm.size()) {
            //     window.alert(S('NothingToBeSaved'));
            //     params.preventDefault = true;
            // }
        }
        else if (type === 'symbolViewSaving' || type === 'symbolViewNewNameInputing') {
            if (type === 'symbolViewSaving') {
                // Symbol 缩略图仍自动生成，但不保存外部预览或快照地址。
                params.symbolView.dm.a('previewURL', undefined);
                params.symbolView.dm.a('snapshotURL', undefined);
            }
            // if (!params.symbolView.dm.size()) {
            //     window.alert(S('NothingToBeSaved'));
            //     params.preventDefault = true;
            // }
        }
        else if (type === 'paste') {
            params.datas.forEach(function(data) {
                var dataBindings = data.getDataBindings();
                if (dataBindings) {
                    // update attrs
                    for (var name in dataBindings.a) {
                        var db = dataBindings.a[name];
                        db.id = db.id + '_copied';
                    }
                    // update styles
                    for (var name in dataBindings.s) {
                        var db = dataBindings.s[name];
                        db.id = db.id + '_copied';
                    }
                    // update properties
                    for (var name in dataBindings.p) {
                        var db = dataBindings.p[name];
                        db.id = db.id + '_copied';
                    }
                }
            });
        }
        else if (type === 'fileRenaming' ||
                 type === 'fileMoving' ||
                 type === 'fileDeleting') {
            // Prevent some files from being renamed, moved or deleted
            if (params.url === 'symbols/basic/ht.json' ||
                params.url === 'symbols/basic' ||
                params.url === 'displays/basic') {
                params.preventDefault = true;
            }
        }
    };

})();






















