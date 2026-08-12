window.isPracticing = window.location.host.indexOf('hightopo') >= 0;

window.hteditor_config = {
    locale: 'zh',
    // color_select: 'red',
    // explorerMode: 'accordion',
    // scenesVisible: true,
    // modelsVisible: true,
    // hoverGuideVisible: false,
    // anchorVisible: false,
    // connectGuideVisible: false,
    // guideLineVisible: false,
    // moveDummyThreshold: false,
    // smartGuideThreshold: 8,
    componentsVisible: !isPracticing,
    displaysEditable: !isPracticing,
    symbolsEditable: !isPracticing,
    componentsEditable: !isPracticing,
    assetsEditable: !isPracticing,
    locateFileEnabled: !isPracticing,
    fontPreview: '图扑软件 - Hightopo',
    expandedTitles: {
        TitleExtension: false
    },
    subConfigs: [
        'custom/configs/config-utils.js',
        'custom/configs/config-points.js',
        'custom/configs/config-inspectorTab.js',
        'custom/configs/config-handleEvent.js',
        'custom/configs/config-valueTypes.js',
        'custom/configs/config-dataBindings.js',
        'custom/configs/config-dataBindingsForSymbol.js',
        'custom/configs/config-connectActions.js',
        'custom/configs/config-inspectorFilter.js',
        'custom/configs/config-customProperties.js',
        'custom/configs/config-onEditorCreated.js',
        'custom/configs/config-onTitleCreating.js',
        'custom/configs/config-onTitleCreated.js',
        'custom/configs/config-onMainToolbarCreated.js',
        'custom/configs/config-onMainMenuCreated.js',
        'custom/configs/config-onRightToolbarCreated.js'
    ],
    libs: [
        'custom/libs/echarts.js'
    ],
    handleInsertSceneFileToDisplayView: function(displayView, fileNode, point) {
        var node = new ht.Node();
        node.a('sceneURL', fileNode.url);
        node.setImage('symbols/html/scene.json')
        if (point) {
            node.p(point.x, point.y);
        }
        else {
            var rect = displayView.graphView.getViewRect();
            if (rect) {
                node.p(rect.x + rect.width/2, rect.y + rect.height/2);
            }
        }
        node.setDisplayName(hteditor.fileNameToDisplayName(fileNode.url));
        displayView.addData(node);
    },
    handleInsertModelFileToDisplayView: function(displayView, fileNode, point) {
        var node = new ht.Node();
        node.a('modelURL', fileNode.url);
        node.setImage('symbols/html/obj.json')
        if (point) {
            node.p(point.x, point.y);
        }
        else {
            var rect = displayView.graphView.getViewRect();
            if (rect) {
                node.p(rect.x + rect.width/2, rect.y + rect.height/2);
            }
        }
        node.setDisplayName(hteditor.fileNameToDisplayName(fileNode.url));
        displayView.addData(node);
    }
};
