(function() {

    window.hteditor_config.onEditorCreated = function(editor, params) {
        // 平台场景是唯一作品边界，固定入口只用于内部加载，不向用户暴露 HT 图纸管理。
        if (editor.displaysTab && editor.displaysTab.setVisible) {
            editor.displaysTab.setVisible(false);
        }
        if (editor.mainMenu && editor.mainMenu.setItems) {
            editor.mainMenu.setItems([]);
        }
        if (editor.mainToolbar && editor.mainToolbar.setItemVisible) {
            editor.mainToolbar.setItemVisible('menu', false);
        }
        [editor.displays && editor.displays.tree,
         editor.displays && editor.displays.list,
         editor.displays && editor.displays.accordion].forEach(function(view) {
            if (view && view.menu && view.menu.setItems) {
                view.menu.setItems([]);
            }
        });
        if (window.InduForgeAssets) {
            window.InduForgeAssets.mount(editor, '2d');
        }
        // editor.leftTopTabView.select(editor.assetsTab);
        // editor.displays.list.menu.setItemVisible('rename', false);
        // editor.symbols.list.menu.setItemVisible('rename', false);
        // editor.components.list.menu.setItemVisible('rename', false);
        // editor.assets.list.menu.setItemVisible('rename', false);

        // Add print item
        // addPrintSelectionItem(editor.displays.tree, 'editor.displays.tree');
        // addPrintSelectionItem(editor.displays.list, 'editor.displays.list');
        // addPrintSelectionItem(editor.symbols.tree, 'editor.symbols.tree');
        // addPrintSelectionItem(editor.symbols.list, 'editor.symbols.list');
        // addPrintSelectionItem(editor.components.tree, 'editor.components.tree');
        // addPrintSelectionItem(editor.components.list, 'editor.components.list');
        // addPrintSelectionItem(editor.assets.tree, 'editor.assets.tree');
        // addPrintSelectionItem(editor.assets.list, 'editor.assets.list');
        // addPrintSelectionItem(editor.mainTabView, 'editor.mainTabView');

        // Add points tab
        // addPointsTab(editor);

        // Add a custom tab to show more information
        // addInspectorTab(editor);

        // Draw extra icon on file list
        // var fileList = editor.displays.list;
        // fileList.addTopPainter(function(g) {
        //     var htIcon = ht.Default.getImage('symbols/可视化/basic/ht.json');
        //     fileList.getDataModel().each(function(file) {
        //         if (fileList.isVisible(file)) {
        //             if (fileList.getLayoutType() === 'list') {
        //                 var x = 0;
        //                 var y = file.p().y - fileList.getRowHeight() / 2;
        //                 var width = fileList.getWidth();
        //                 var height = fileList.getRowHeight();
        //                 g.fillStyle = 'yellow';
        //                 g.beginPath();
        //                 g.rect(width - 16, y, 16, 16);
        //                 g.fill();
        //                 ht.Default.drawStretchImage(g, htIcon, 'uniform', width - 16, y, 16, 16);
        //             }
        //             else {
        //                 var rect = file.getRect();
        //                 g.fillStyle = 'yellow';
        //                 g.beginPath();
        //                 g.rect(rect.x + rect.width - 16, rect.y, 16, 16);
        //                 g.fill();
        //                 ht.Default.drawStretchImage(g, htIcon, 'uniform', rect.x + rect.width - 16, rect.y, 16, 16);
        //             }
        //         }
        //     });
        // });
    };

})();





















