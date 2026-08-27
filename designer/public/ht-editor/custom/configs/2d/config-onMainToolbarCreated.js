(function() {

    window.hteditor_config.onMainToolbarCreated = function(editor) {

        var mainToolbar = editor.mainToolbar;

        // Hide some items from main toolbar
        mainToolbar.setItemVisible('Star', false);
        mainToolbar.setItemVisible('star', false);
        mainToolbar.setItemVisible('Triangle', false);
        mainToolbar.setItemVisible('triangle', false);

        // remove items from main toolbar
        mainToolbar.removeItemById('Node');
        mainToolbar.removeItemById('Group');
        mainToolbar.removeItemById('SubGraph');
        mainToolbar.removeItemById('Edge');

        // Add some items to main toolbar
        addItemsForDisplay(editor);
        addItemsForSymbol(editor);
    };

    function addItemsForDisplay(editor) {
        var mainToolbar = editor.mainToolbar;
        var S = hteditor.getString;

        var id, toolTip, icon, type, initData;

        mainToolbar.addItem({
            separator: true,
            visible: function() {
                return !!editor.displayView;
            }
        });

        id = 'Edge';
        toolTip = S('editor.edge');
        icon = 'editor.display.edge';
        type = ht.Edge;
        initData = function (data) {
            data.s({
                'edge.color': '#009687',
                'edge.width': 1
            });
        };
        mainToolbar.addItem(editor.createDisplayItem(id, toolTip, icon, type, initData));

        var autoRouteItem = typeof hteditor.createItem === 'function'
            ? hteditor.createItem('AutoRoute', S('AutoRoute'), 'editor.changepath')
            : { id: 'AutoRoute', toolTip: S('AutoRoute'), icon: 'editor.changepath' };
        autoRouteItem.visible = function() { return !!editor.displayView; };
        autoRouteItem.action = function() { autoRoute(editor); };
        mainToolbar.addItem(autoRouteItem);

        id = 'Pipe';
        toolTip = S('Pipe');
        icon = 'custom/images/pipe.json';
        type = ht.Shape;
        initData = function(data) {
            data.setDisplayName(S('Pipe'));
            data.a('induforge.path.type', 'pipe');
            data.a({
                'induforge.pipe.flowMode': 'continuous',
                'induforge.pipe.running': true,
                'induforge.pipe.reverse': false,
                'induforge.pipe.speed': 2,
                'induforge.pipe.elementSize': 12,
                'induforge.pipe.spacing': 36,
                'induforge.pipe.count': 6
            });
            data.s({
                'shape.background': null,
                'shape.border.width': 12,
                'shape.border.color': '#56615C',
                'shape.border.cap': 'round',
                'shape.border.join': 'round',
                'shape.dash': true,
                'shape.dash.pattern': [100000, 0],
                'shape.dash.width': 6,
                'shape.dash.color': '#47CFA0',
                'shape.dash.flow': false,
                'shape.dash.flow.reverse': false,
                'shape.dash.flow.step': 2
            });
            if (window.InduForgePipe) window.InduForgePipe.apply(data);
        };
        mainToolbar.addItem(editor.createDisplayItem(id, toolTip, icon, type, initData));

        // id = 'VProgressBar';
        // toolTip = S('VProgressBar');
        // icon = 'custom/images/vprogressbar.json';
        // type = ht.Node;
        // initData = function(data) {
        //     data.setImage('symbols/basic/vProgressBar.json');
        //     data.a({
        //         'pb.value.visible': false,
        //         'pb.fore.color': "#3399FF",
        //         'pb.background': "#38373B"
        //     });
        // };
        // mainToolbar.addItem(editor.createDisplayItem(id, toolTip, icon, type, initData));

        // id = 'HProgressBar';
        // toolTip = S('HProgressBar');
        // icon = 'custom/images/hprogressbar.json';
        // type = ht.Node;
        // initData = function(data) {
        //     data.setImage('symbols/basic/hProgressBar.json');
        //     data.a({
        //         'pb.value.visible': false,
        //         'pb.fore.color': "#3399FF",
        //         'pb.background': "#38373B"
        //     });
        // };
        // mainToolbar.addItem(editor.createDisplayItem(id, toolTip, icon, type, initData));

        // id = 'Table';
        // toolTip = S('Table');
        // icon = 'custom/images/table.json';
        // type = ht.Node;
        // initData = function(data) {
        //     data.setImage('symbols/basic/table.json');
        //     data.a({
        //         'table.background': null,
        //         'table.line.color': '#3399FF'
        //     });
        // };
        // mainToolbar.addItem(editor.createDisplayItem(id, toolTip, icon, type, initData));

        // id = 'Clock';
        // toolTip = S('Clock');
        // icon = 'custom/images/clock.json';
        // type = ht.Node;
        // initData = function(data) {
        //     var date = new Date();
        //     data.setImage('symbols/basic/clock.json');
        //     data.a({
        //         'clock.second': date.getSeconds(),
        //         'clock.minute': date.getMinutes(),
        //         'clock.hour': date.getHours(),
        //         'clock.background': '#38373B',
        //         'clock.minute.color': '#3399FF',
        //         'clock.hour.color': '#3399FF',
        //         'clock.number.color': '#FFFFE0',
        //         'clock.border.color': '#FFCC99',
        //         'clock.scale.color': null

        //     });
        // };
        // mainToolbar.addItem(editor.createDisplayItem(id, toolTip, icon, type, initData));

        // id = 'Pipe';
        // toolTip = S('Pipe');
        // icon = 'custom/images/pipe.json';
        // type = ht.Shape;
        // initData = function(data) {
        //     data.s({
        //         "shape.background": null,
        //         "shape.border.width": 10,
        //         "shape.dash": true,
        //         "shape.dash.pattern": [4, 8],
        //         "shape.dash.3d": true,
        //         "shape.dash.3d.color": "rgb(247,247,247)",
        //         "note.border.color": "rgb(61,61,61)",
        //         "border.color": "#929292",
        //         "shape.border.color": "#60ACFC",
        //         "shape.dash.color": "#929292",
        //         "border.width": 1
        //     });
        // };
        // mainToolbar.addItem(editor.createDisplayItem(id, toolTip, icon, type, initData));
    }

    // 自动绕障只在编辑态调用路由插件，计算结果烘焙为普通 Shape 点集。
    function autoRoute(editor) {
        var graphView = editor.displayView && editor.displayView.graphView;
        var dm = graphView && graphView.dm && graphView.dm();
        var shape = graphView && graphView.sm && graphView.sm().ld();
        if (!(shape instanceof ht.Shape) || !window.AStar) {
            editor.showMessage && editor.showMessage(hteditor.getString('AutoRouteSelectPath'), 'warning');
            return;
        }
        var sourcePoints = shape.getPoints && shape.getPoints();
        var points = sourcePoints && sourcePoints.toArray ? sourcePoints.toArray() : sourcePoints;
        if (!points || points.length < 2) return;
        var gridSize = 20;
        var start = points[0];
        var end = points[points.length - 1];
        var minX = Math.min(start.x, end.x) - 400;
        var minY = Math.min(start.y, end.y) - 400;
        var maxX = Math.max(start.x, end.x) + 400;
        var maxY = Math.max(start.y, end.y) + 400;
        var width = Math.max(2, Math.ceil((maxX - minX) / gridSize));
        var height = Math.max(2, Math.ceil((maxY - minY) / gridSize));
        var matrix = [];
        for (var x = 0; x < width; x++) {
            matrix[x] = [];
            for (var y = 0; y < height; y++) matrix[x][y] = 1;
        }
        dm.each(function(data) {
            if (data === shape || !(data instanceof ht.Node) || !data.getRect) return;
            var rect = data.getRect();
            var fromX = Math.max(0, Math.floor((rect.x - 10 - minX) / gridSize));
            var toX = Math.min(width - 1, Math.ceil((rect.x + rect.width + 10 - minX) / gridSize));
            var fromY = Math.max(0, Math.floor((rect.y - 10 - minY) / gridSize));
            var toY = Math.min(height - 1, Math.ceil((rect.y + rect.height + 10 - minY) / gridSize));
            for (var ix = fromX; ix <= toX; ix++) for (var iy = fromY; iy <= toY; iy++) matrix[ix][iy] = 0;
        });
        var sx = Math.max(0, Math.min(width - 1, Math.round((start.x - minX) / gridSize)));
        var sy = Math.max(0, Math.min(height - 1, Math.round((start.y - minY) / gridSize)));
        var ex = Math.max(0, Math.min(width - 1, Math.round((end.x - minX) / gridSize)));
        var ey = Math.max(0, Math.min(height - 1, Math.round((end.y - minY) / gridSize)));
        matrix[sx][sy] = matrix[ex][ey] = 1;
        var graph = new AStar.Graph(matrix);
        var route = AStar.search(graph, graph.grid[sx][sy], graph.grid[ex][ey], { closest: false });
        if (!route.length) {
            editor.showMessage && editor.showMessage(hteditor.getString('AutoRouteNotFound'), 'warning');
            return;
        }
        var routed = [start];
        route.forEach(function(item) { routed.push({ x: minX + item.x * gridSize, y: minY + item.y * gridSize }); });
        routed[routed.length - 1] = end;
        var compact = routed.filter(function(point, index) {
            if (index === 0 || index === routed.length - 1) return true;
            var previous = routed[index - 1];
            var next = routed[index + 1];
            return !((previous.x === point.x && point.x === next.x) || (previous.y === point.y && point.y === next.y));
        });
        shape.setPoints(new ht.List(compact));
    }

    function addItemsForSymbol(editor) {
        var mainToolbar = editor.mainToolbar;
        var S = hteditor.getString;

        var id, toolTip, icon, type, initComp;

        mainToolbar.addItem({
            separator: true,
            visible: function() {
                return !!editor.symbolView;
            }
        });

        // id = 'vProgressBar';
        // toolTip = S('VProgressBar');
        // icon = 'custom/images/vprogressbar.json';;
        // type = 'comp';
        // initComp = function (comp, rect, click) {
        //     comp.type = 'components/progressBar/progressBar.json';
        //     comp.direction = 'v';
        //     comp.valueVisible = false;
        //     comp.foreColor = '#3399FF';
        //     comp.background = '#38373B';
        //     if (click) {
        //         comp.rect[2] = 10;
        //         comp.rect[3] = 30;
        //     }
        // };
        // mainToolbar.addItem(editor.createSymbolItem(id, toolTip, icon, type, initComp));

        // id = 'hProgressBar';
        // toolTip = S('HProgressBar');
        // icon = 'custom/images/hprogressbar.json';
        // type = 'comp';
        // initComp = function (comp, rect, click) {
        //     comp.type = 'components/progressBar/progressBar.json';
        //     comp.direction = 'h';
        //     comp.valueVisible = false;
        //     comp.foreColor = '#3399FF';
        //     comp.background = '#38373B';
        //     if (click) {
        //         comp.rect[2] = 30;
        //         comp.rect[3] = 10;
        //     }
        // };
        // mainToolbar.addItem(editor.createSymbolItem(id, toolTip, icon, type, initComp));

        id = 'star';
        toolTip = S('editor.star');
        icon = 'editor.symbol.star';
        type = 'star';
        initComp = function(comp, rect, click) {
            comp.background = '#3399FF';
            comp.borderWidth = 1;
            comp.borderColor = '#38373B';
            comp.gradient = 'linear.southwest';
        };
        mainToolbar.addItem(editor.createSymbolItem(id, toolTip, icon, type, initComp));

    }

})();
