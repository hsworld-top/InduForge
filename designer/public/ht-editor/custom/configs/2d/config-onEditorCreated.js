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
        if (window.InduForgeBindings) {
            window.InduForgeBindings.mount(editor);
        }
        installShapeBoundsSync();
        installCurrentGraphViewHooks(editor);
        if (editor.addEventListener && !editor.__induforgeShapeEditingListenerInstalled) {
            editor.__induforgeShapeEditingListenerInstalled = true;
            editor.addEventListener(function(event) {
                if (event && (event.type === 'displayViewCreated' || event.type === 'displayViewOpened')) {
                    var displayView = event.params && event.params.displayView;
                    installGraphViewHooks(displayView && displayView.graphView);
                }
            });
        }
        scheduleGraphViewHookProbe(editor, 0);
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

    var graphViewHookProbeDelays = [0, 50, 200, 1000];

    // 编辑器创建事件早于作品视图创建；事件监听是主路径，有限探测用于覆盖异步打开时序。
    function scheduleGraphViewHookProbe(editor, index) {
        if (!window.setTimeout || index >= graphViewHookProbeDelays.length) return;
        window.setTimeout(function() {
            if (installCurrentGraphViewHooks(editor)) return;
            scheduleGraphViewHookProbe(editor, index + 1);
        }, graphViewHookProbeDelays[index]);
    }

    function installCurrentGraphViewHooks(editor) {
        return installGraphViewHooks(editor.displayView && editor.displayView.graphView);
    }

    function installShapeBoundsSync() {
        var prototype = ht.Shape && ht.Shape.prototype;
        if (!prototype) return;
        if (prototype.fireShapeChange && !prototype.__induforgeShapeBoundsSyncInstalled) {
            prototype.__induforgeShapeBoundsSyncInstalled = true;
            var originalFireShapeChange = prototype.fireShapeChange;
            prototype.fireShapeChange = function() {
                var result = originalFireShapeChange.apply(this, arguments);
                ensureShapeRectMatchesPoints(this);
                return result;
            };
        }
        if (prototype.setPosition && !prototype.__induforgeShapePositionSyncInstalled) {
            prototype.__induforgeShapePositionSyncInstalled = true;
            var originalSetPosition = prototype.setPosition;
            prototype.setPosition = function() {
                var previousPosition = this.getPosition && this.getPosition();
                var previousPoints = copyShapePoints(this);
                var result = originalSetPosition.apply(this, arguments);
                if (this._55I || !previousPosition || !previousPoints || !this.shiftPoints) return result;
                var currentPosition = this.getPosition && this.getPosition();
                var offsetX = currentPosition && currentPosition.x - previousPosition.x;
                var offsetY = currentPosition && currentPosition.y - previousPosition.y;
                if ((!offsetX && !offsetY) || !shapePointsEqual(previousPoints, copyShapePoints(this))) return result;

                // 当前 HT 编辑构建会只更新 Shape 外框；补齐点集平移后，整体移动手柄才能带动路径。
                this.shiftPoints(offsetX, offsetY);
                return result;
            };
        }
    }

    function copyShapePoints(shape) {
        if (!shape.getPoints) return null;
        var sourcePoints = shape.getPoints();
        var points = sourcePoints && sourcePoints.toArray ? sourcePoints.toArray() : sourcePoints;
        if (!points || !points.length) return null;
        return points.map(function(point) { return { x: point.x, y: point.y }; });
    }

    function shapePointsEqual(left, right) {
        if (!left || !right || left.length !== right.length) return false;
        return left.every(function(point, index) {
            return point.x === right[index].x && point.y === right[index].y;
        });
    }

    function installGraphViewHooks(graphView) {
        var dataModel = graphView && graphView.dm && graphView.dm();
        if (!graphView || !dataModel) return false;
        if (!graphView.__induforgeShapeEditingInstalled) {
            graphView.__induforgeShapeEditingInstalled = true;
            if (graphView.enableDashFlow) {
                graphView.enableDashFlow(50);
            }
            if (graphView.enableFlow) {
                graphView.enableFlow(50);
            }
            configureShapeMoveHandle(graphView);
            installShapeHitTolerance(graphView);
            if (graphView.addInteractorListener) {
                graphView.addInteractorListener(function(event) {
                    if (event && event.kind === 'clickData' && event.data instanceof ht.Shape) {
                        leaveShapePointEditing(graphView);
                    }
                    if (event && event.kind === 'doubleClickData' && event.data instanceof ht.Shape) {
                        enterShapePointEditing(graphView);
                    }
                    if (event && (event.kind === 'beginEditPoint' ||
                                  event.kind === 'betweenEditPoint' ||
                                  event.kind === 'selectPoint')) {
                        var selectedShape = graphView.sm && graphView.sm().ld();
                        if (selectedShape instanceof ht.Shape) {
                            refreshShapeBounds(graphView, selectedShape, event.kind === 'selectPoint');
                        }
                    }
                });
            }
            if (dataModel.addDataModelChangeListener) {
                dataModel.addDataModelChangeListener(function(event) {
                    if (event && event.data instanceof ht.Shape) {
                        refreshShapeBounds(graphView, event.data, true);
                    }
                });
            }
            if (dataModel.addDataPropertyChangeListener) {
                dataModel.addDataPropertyChangeListener(function(event) {
                    var property = event && event.property;
                    if (event && event.data instanceof ht.Shape && (property === 'points' || property === 'segments')) {
                        refreshShapeBounds(graphView, event.data, true);
                    }
                    if (event && event.data && property && property.indexOf('induforge.pipe.') === 0 && window.InduForgePipe) {
                        window.InduForgePipe.apply(event.data);
                    }
                });
            }
            var selectionModel = dataModel.sm && dataModel.sm();
            if (selectionModel && selectionModel.addSelectionChangeListener) {
                selectionModel.addSelectionChangeListener(function() {
                    leaveShapePointEditing(graphView);
                    var data = selectionModel.ld && selectionModel.ld();
                    if (data instanceof ht.Shape) {
                        refreshShapeBounds(graphView, data, true);
                    }
                });
            }
        }
        refreshAllShapeBounds(graphView, dataModel);
        return true;
    }

    // HT 默认仅为很小的对象显示整体移动手柄；路径和管道需要始终提供明确的四向拖拽入口。
    function configureShapeMoveHandle(graphView) {
        if (!graphView.setEditStyle) return;
        graphView.setEditStyle('anchorVisible', false);
        graphView.setEditStyle('moveDummyThreshold', true);
        graphView.setEditStyle('moveDummySensitivity', 18);
        graphView.setEditStyle('moveDummyPosition', [0, 0, -20, -20]);
        installShapeMoveHandleBehavior(graphView);
    }

    function installShapeMoveHandleBehavior(graphView) {
        var editInteractor = graphView.getEditInteractor && graphView.getEditInteractor();
        var moveHandle = editInteractor && editInteractor.getSubModule && editInteractor.getSubModule('MoveDummy');
        if (!moveHandle || !moveHandle.startEdit || !moveHandle._46O || moveHandle.__induforgeBehaviorInstalled) return;
        moveHandle.__induforgeBehaviorInstalled = true;
        var originalStartEdit = moveHandle.startEdit;
        var originalEndEdit = moveHandle._46O;

        moveHandle.startEdit = function(interactor, event) {
            if (!graphView.__induforgeShapeMoveHandleDragging) {
                graphView.__induforgeShapeMoveHandleDragging = true;
                graphView.__induforgeShapeMoveHandleAutoScroll = graphView.autoScroll;
                // 整体移动手柄只改变元素；靠近画布边缘时也不能触发 HT 的画布自动滚动。
                graphView.autoScroll = function() { return { x: 0, y: 0 }; };
            }
            if (interactor.setCursor) interactor.setCursor('move');
            try {
                return originalStartEdit.apply(this, arguments);
            } catch (error) {
                restoreShapeMoveHandleAutoScroll(graphView);
                throw error;
            }
        };
        moveHandle._46O = function() {
            try {
                return originalEndEdit.apply(this, arguments);
            } finally {
                restoreShapeMoveHandleAutoScroll(graphView);
            }
        };
    }

    function restoreShapeMoveHandleAutoScroll(graphView) {
        if (!graphView.__induforgeShapeMoveHandleDragging) return;
        graphView.autoScroll = graphView.__induforgeShapeMoveHandleAutoScroll;
        delete graphView.__induforgeShapeMoveHandleAutoScroll;
        delete graphView.__induforgeShapeMoveHandleDragging;
    }

    function refreshAllShapeBounds(graphView, dataModel) {
        if (!dataModel.each) return;
        dataModel.each(function(data) {
            if (data instanceof ht.Shape) {
                refreshShapeBounds(graphView, data, true);
            }
        });
    }

    // 单击保留 HT 的整体移动/缩放模式，双击后才进入路径控制点编辑，避免两种操作互相覆盖。
    function enterShapePointEditing(graphView) {
        var editInteractor = graphView.getEditInteractor && graphView.getEditInteractor();
        if (editInteractor && !editInteractor.pointsEditingMode) {
            editInteractor.pointsEditingMode = true;
        }
    }

    function leaveShapePointEditing(graphView) {
        var editInteractor = graphView.getEditInteractor && graphView.getEditInteractor();
        if (editInteractor && editInteractor.pointsEditingMode) {
            editInteractor.pointsEditingMode = false;
            if (graphView.invalidateSelection) graphView.invalidateSelection();
        }
    }

    // 反序列化期间 HT 会暂时抑制 Shape 边界计算，因此需要在当前调用栈结束后再刷新一次。
    function refreshShapeBounds(graphView, shape, deferred) {
        if (shape.fireShapeChange) shape.fireShapeChange();
        if (graphView.invalidateData) graphView.invalidateData(shape);
        if (!deferred || !window.setTimeout || shape.__induforgeShapeBoundsTimer !== undefined) return;
        shape.__induforgeShapeBoundsTimer = window.setTimeout(function() {
            delete shape.__induforgeShapeBoundsTimer;
            if (shape.fireShapeChange) shape.fireShapeChange();
            if (graphView.invalidateData) graphView.invalidateData(shape);
        }, 0);
    }

    // 当前 HT 编辑态会抑制 fireShapeChange 内部的边界同步；按其原生算法重建点集外接矩形。
    function ensureShapeRectMatchesPoints(shape) {
        if (!shape.getPoints || !shape.getRect || !shape.setRect) return;
        var sourcePoints = shape.getPoints();
        var points = sourcePoints && sourcePoints.toArray ? sourcePoints.toArray() : sourcePoints;
        if (!points || !points.length) return;
        var minX = Infinity;
        var minY = Infinity;
        var maxX = -Infinity;
        var maxY = -Infinity;
        points.forEach(function(point) {
            if (!point || !isFinite(point.x) || !isFinite(point.y)) return;
            minX = Math.min(minX, point.x);
            minY = Math.min(minY, point.y);
            maxX = Math.max(maxX, point.x);
            maxY = Math.max(maxY, point.y);
        });
        if (!isFinite(minX) || !isFinite(minY) || !isFinite(maxX) || !isFinite(maxY)) return;
        var rect = shape.getRect();
        var expected = { x: minX, y: minY, width: maxX - minX, height: maxY - minY };
        if (rect && rect.x === expected.x && rect.y === expected.y &&
            rect.width === expected.width && rect.height === expected.height) return;

        // _55I 是当前 HT Shape 在 setRect 时阻止反向平移点集的内部保护位。
        var previousUpdating = shape._55I;
        shape._55I = 1;
        shape.setRect(expected);
        if (previousUpdating === undefined) delete shape._55I;
        else shape._55I = previousUpdating;
    }

    function installShapeHitTolerance(graphView) {
        if (!graphView || !graphView.getDataAt || graphView.__induforgeShapeHitToleranceInstalled) return;
        graphView.__induforgeShapeHitToleranceInstalled = true;
        var originalGetDataAt = graphView.getDataAt;
        graphView.getDataAt = function(point, filter, tolerance) {
            var data = originalGetDataAt.call(this, point, filter, tolerance);
            if (data || (typeof tolerance === 'number' && tolerance >= 8)) return data;
            return originalGetDataAt.call(this, point, function(candidate) {
                return candidate instanceof ht.Shape && (!filter || filter(candidate));
            }, 8);
        };
    }

})();
