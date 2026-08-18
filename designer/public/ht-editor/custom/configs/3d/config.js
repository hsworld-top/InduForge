window.hteditor_config = {
    locale: 'zh',
    promptBeforeClosing: false,
    indent: 110,
    imageSize: 400,
    numberPrecision: {
        position: 0,
        scale: 3,
        anchor: 4,
        rotation: 0,
        size: 0
    },
    importConfirm: true,
    locateFileEnabled: false,
    checkForFileChanges: false,
    newIfFailToOpen: false,

    // 保留资源路径前缀配置入口
    // urlPrefix: '2d/',

    // 配置编辑器扩展，运行于 client.js 加载前
    subConfigs: [
        'custom/configs/3d/config-onEditor3dCreated.js',

        'custom/configs/config-valueTypes.js',
        'custom/configs/config-dataBindings.js'
    ],

    // 配置运行时依赖类库，运行于 client.js 加载后，
    // 一般放置需要与运行时共享的通用类库
    libs: [
        'custom/libs/echarts.js',
        'custom/libs/InduForgeService.js',
        'vs/loader.js',
        'vs/editor/editor.main.js'
    ],
    serviceClass:"InduForgeService",
    // add by jie.li 2021-6-30 bug 12390 3D模型编辑器退出保存场景名可以为空格
    saveFileName: function (fileName, url) {

        if (!fileName) return false;
        fileName = fileName.trim();
        if (!fileName) return false;
        if (/[!@?#$%^&*/]/.test(fileName)) return false;
        return fileName;
    }
    // end
};
