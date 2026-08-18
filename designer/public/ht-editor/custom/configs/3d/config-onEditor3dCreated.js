(function () {
    window.hteditor_config.onEditor3dCreated = function (editor3d) {
        // 3D 原生作品管理被平台场景和工程资源库取代。
        if (editor3d.mainMenu && editor3d.mainMenu.setItems) {
            editor3d.mainMenu.setItems([]);
        }
        if (editor3d.mainToolbar && editor3d.mainToolbar.setItemVisible) {
            editor3d.mainToolbar.setItemVisible('menu', false);
        }
        [editor3d.scenes, editor3d.models, editor3d.symbols, editor3d.materials, editor3d.assets].forEach(function(explorer) {
            if (!explorer) return;
            [explorer.tree, explorer.list, explorer.accordion].forEach(function(view) {
                if (view && view.menu && view.menu.setItems) {
                    view.menu.setItems([]);
                }
            });
        });
        var tabs = editor3d.leftTopTabView && editor3d.leftTopTabView.getTabModel().getDatas();
        if (tabs) {
            tabs.each(function(tab) {
                // 3D 场景入口和 Provider 原生资源均由平台管理，不暴露 HT 作品管理页签。
                if (tab.setVisible) {
                    tab.setVisible(false);
                }
            });
        }
        if (window.InduForgeAssets) {
            window.InduForgeAssets.mount(editor3d, '3d');
        }
    };
})();
