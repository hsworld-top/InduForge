window.hteditor_config = {
    // color_select: '#FF7733',
    // explorerMode: 'accordion',
    locale: 'zh',
    // rotateAsClock: true,
    // filterDisplayViewEnabled: true,
    // filterSymbolViewEnabled: true,
    // mixMaskAndBackground: false,
    // saveSymbolCustomPropertyDefaultValue: true,
    // saveCompCustomPropertyDefaultValue: true,
    // importDataBindingsButtonVisible: false,
    // scenesVisible: true,
    // modelsVisible: true,
    // fitContentForDisplayView: true,
    // fitContentForSymbolView: true,
    promptBeforeClosing: false,
    symbolsVisible: false,
    componentsVisible: false,
    assetsVisible: false,
    displaysEditable: false,
    symbolsEditable: false,
    componentsEditable: false,
    assetsEditable: false,
    newIfFailToOpen: false,
    locateFileEnabled: false,
    checkForFileChanges: false,
    fontPreview: 'InduForge',
    expandedTitles: {
        TitleExtension: false
    },
    subConfigs: [
        'custom/configs/2d/config-utils.js',
        'custom/configs/2d/config-inspectorTab.js',
        'custom/configs/2d/config-handleEvent.js',
        'custom/configs/config-valueTypes.js',
        'custom/configs/config-dataBindings.js',
        'custom/configs/2d/config-connectActions.js',
        'custom/configs/2d/config-inspectorFilter.js',
        'custom/configs/config-customProperties.js',
        'custom/configs/2d/config-onEditorCreated.js',
        'custom/configs/2d/config-onMainToolbarCreated.js',
        'custom/configs/2d/config-onRightToolbarCreated.js'
    ],
    libs: [
        'custom/libs/echarts.js',
        'vs/loader.js',
        'custom/libs/InduForgeService.js',
        'vs/editor/editor.main.nls.js',
        'vs/editor/editor.main.js'
    ],
    handleInsertSceneFileToGraphView: function(displayView, fileNode, point) {
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
    handleInsertModelFileToGraphView: function(displayView, fileNode, point) {
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
    },
    serviceClass:"InduForgeService"
};
