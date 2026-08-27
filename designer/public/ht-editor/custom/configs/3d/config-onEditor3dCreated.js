(function () {
    function hidePlatformManagedSceneProperties(editor3d) {
        var inspector = editor3d.sceneInspector;
        if (!inspector || inspector.__induforgePlatformPropertiesHidden) return;

        var originalIsPropertyVisible = inspector.isPropertyVisible;
        inspector.isPropertyVisible = function(row) {
            var name = row && row.keys && row.keys.name;
            if (name === 'previewURL' || name === 'snapshotURL') {
                return false;
            }
            return originalIsPropertyVisible.call(this, row);
        };
        inspector.__induforgePlatformPropertiesHidden = true;
        if (inspector.filterProperties) {
            inspector.filterProperties();
        }
    }

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
        if (window.InduForgeBindings) {
            window.InduForgeBindings.mount(editor3d);
        }
        hidePlatformManagedSceneProperties(editor3d);
        if (editor3d.addEventListener) {
            editor3d.addEventListener(function(event) {
                if (event.type !== 'sceneSaving') return;
                // Viewer 入口和缩略图均由平台管理，保存时清除旧场景遗留值。
                editor3d.dm.a('previewURL', undefined);
                editor3d.dm.a('snapshotURL', undefined);
            });
        }
    };
})();
