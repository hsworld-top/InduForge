function init() {
  window.isonup = true;
  window.zoomimageall = false;
  var isFast = false;
  // 移动端设备物理像素与CSS像素分辨率的比率
  window.KM_PixelRatio = (function () {
    var canvas = window.document.createElement('canvas'),
      context = canvas.getContext('2d'),
      backingStore = context.backingStorePixelRatio ||
        context.webkitBackingStorePixelRatio ||
        context.mozBackingStorePixelRatio ||
        context.msBackingStorePixelRatio ||
        context.oBackingStorePixelRatio ||
        context.backingStorePixelRatio || 1;

    return (window.devicePixelRatio || 1) / backingStore;
  })();
  // 自定义鹰眼定义
  var ToggleOverview = function (graphView) {
    var self = this;
    ToggleOverview.superClass.constructor.apply(self, [graphView]);
    self._expand = true;
    let div = document.createElement('div');
    div.setAttribute('id', 'overviewButton');
    div.style.setProperty('width', '24px', null);
    div.style.setProperty('height', '24px', null);
    div.style.setProperty('position', 'absolute', null);
    div.style.setProperty('left', '0', null);
    div.style.setProperty('top', '0', null);
    div.style.setProperty('background', 'url(./maximize.png) no-repeat', null);
    div.style.setProperty('background-position', 'center center', null);
    self._view.appendChild(div);

    function handleTransitionEnd(e) {
      if (e.propertyName === 'width') {
        self.invalidate();
      }
    }

    self._view.addEventListener('webkitTransitionEnd', handleTransitionEnd, false);
    self._view.addEventListener('transitionend', handleTransitionEnd, false);
    let eventName = ht.Default.isTouchable ? 'touchstart' : 'mousedown';
    function handleClick(e) {
      if (self._expand) {
        self._view.style.setProperty('width', '24px', null);
        self._view.style.setProperty('height', '24px', null);
        self._canvas.style.setProperty('opacity', '0', null);
        self._mask.style.setProperty('opacity', '0', null);
        div.style.setProperty('background-image', 'url(./maximize.png)', null);
        div.style.setProperty('width', '24px', null);
        div.style.setProperty('height', '24px', null);
        self._expand = false;
      } else {
        self._view.style.setProperty('width', '', null);
        self._view.style.setProperty('height', '', null);
        self._canvas.style.setProperty('opacity', '1', null);
        self._mask.style.setProperty('opacity', '1', null);
        div.style.setProperty('background-image', 'url(./maximize.png)', null);
        div.style.setProperty('width', '24px', null);
        div.style.setProperty('height', '24px', null);
        self._expand = true;
      }
      self.invalidate();
      e.stopPropagation();
    }
    div.addEventListener(eventName, handleClick);
    div.addEventListener('click', handleClick);
    self.setContentBackground('white');
  }
  ht.Default.def(ToggleOverview, ht.graph.Overview, {});

  // 图纸
  graphView = new ht.graph.GraphView();
  // 鹰眼
  overview = new ToggleOverview(graphView);
  overview.getView().setAttribute('id', 'htoverview');
  overview.getView().className = 'overview overviewAnim';
  // 分别添加到页面
  graphView.addToDOM();
  document.body.appendChild(overview.getView());
  // 页面刷新时调用图纸刷新
  window.addEventListener('resize', function (e) {
    graphView.invalidate();
    // add by jingyu.liu shuai.zhao  for bug 11876 【神东项目0125定制版本】手机端端底图缩放时卡顿，不够流畅 2021-05-28
    canvaswidth = canvas.width;
    canvasheight = canvas.height;
    setTimeout(function () {
      Promise.all([
        createImageBitmap(canvas, 0, 0, canvas.clientWidth * KM_PixelRatio, canvas.clientHeight * KM_PixelRatio)
      ]).then(function (sprites) {
        img = sprites[0];
        imgzoom = 1;
      })
    }, 20);
    // end
  }, false);

  var display = window.location.search.split('&')[0].split('tag=displays')[1];
  let allNodeArr = [
    // '人员',
    '领导',
    '员工',
    '外委队',
    // '车辆',
    '皮卡指挥车',
    '材料车',
    '工程车',
    '运人车',
    '指挥车',
    '特种车',
    '防暴车',
    '低污染车',
    '基站',
    // '安全测点',
    '瓦斯',
    '烟雾',
    '温度',
    '一氧化碳',
    '二氧化碳',
    '风速',
    '压力',
    '水位',
    "综合测点",
    "安全其他",
    // '生产设备',
    '皮带电机',
    '风门',
    '风筒',
    '排水泵',
    '采掘设备',
    "工作面",
    "采煤机",
    "转载机",
    "破碎机",
    "电机",
    "变电所",
    "掘锚机",
    "连运机",
    "梭车",
    "锚杆机",
    "水泵房",
    "离心泵",
    //摄像头
    '摄像头'
  ];
  window.globalGraph = new Graph(0);
  var baseStationArr = [];
  // 计算Edge路径总长, 便于设置Graph中权重, 规划最短路径
  function calcPathDistance(arr) {
    var res = 0;
    for (let m = 0; m < arr.length - 1; m++) {
      res += Math.sqrt(Math.pow(arr[m + 1].y - arr[m].y, 2) + Math.pow(arr[m + 1].x - arr[m].x, 2))
    }
    return res;
  }

  graphView.deserialize('displays/' + decodeURI(display), function (json, dm, gv, datas) {
  setTimeout(() => {
    graphView.fitContent({
      duration: 200,
      finishFunc: function () {
        window.beginZoomRange = graphView.getZoom();
        updateCurrentZoomToKP(graphView.getZoom())
      }
    })
    var nameNew = [];
    // 基站数组初始化数据
    graphView.getDataModel().each(node => {

      node.s('pixelPerfect', true);
      node.s('preventDefaultWhenInteractive', false)
      var iconBelongLayer = "";
      if (node.getImage && node.getImage().indexOf('symbols') !== -1) {
        if(ht.Default.getImage(node.getImage())){
          var iconBelongLayer = ht.Default.getImage(node.getImage())["belongLayer"]
        }else{
          iconBelongLayer =  node.a('belongLayer')
        }
        if(iconBelongLayer){
            node.setScale(1, 1)
            if (iconBelongLayer.indexOf('基站') !== -1 && node.getEdges() !== undefined && baseStationArr.indexOf(node.getDisplayName()) === -1) {
              baseStationArr.push(node.getDisplayName());
            }
        }
      }
      // 对不同图元设置缓存机制
      // if (node.getImage && node.getImage().indexOf('symbols') !== -1 &&node.getImage().indexOf("基站")===-1 && node.getImage().indexOf("摄像头") === -1) {
      //   if (nameNew.length === 0) {
      //     nameNew.push(node.getImage())
      //     ht.Default.setImageCacheRule(node.getImage(), true)
      //   } else {
      //     nameNew.forEach(data => {
      //       if (!nameNew.includes(node.getImage())) {
      //         nameNew.push(node.getImage())
      //         ht.Default.setImageCacheRule(node.getImage(), true)
      //       }
      //     });
      //   }
      // }
    });
    globalGraph.initVertex(baseStationArr);
    // 默认不显示任何的Edge连线
    graphView.getDataModel().each(node => {
      if (node.getClassName() === 'ht.Edge') {
        node.s('2d.visible', false);
        let middlePoints = node.s('edge.points') ? node.s('edge.points')._as : [];
        let startPoint = node.getSource().getPosition();
        let endPoint = node.getTarget().getPosition();
        let allPointsArray = [];
        allPointsArray.push(startPoint);
        allPointsArray = allPointsArray.concat(middlePoints);
        allPointsArray.push(endPoint);
        let weight = calcPathDistance(allPointsArray);
        globalGraph.addEdge(node.getSource().getDisplayName(), node.getTarget().getDisplayName(), weight);
      }
    });

    // 图层默认显示所有
    allNodeArr.forEach(item => {
      dm.a(item, true)
    })
    // graphView.fitContent(false, 20, false);
    // 动画启用
    graphView.getDataModel().enableAnimation();

    // 添加图纸右上角自定义按钮
    var layerButton = new ht.widget.Button();
    layerButton.setLabel('');
    layerButton.setToolTip('图层');
    layerButton.enableToolTip();
    layerButton.setIcon('images/layer.png');
    layerButton.setWidth(24);
    layerButton.setHeight(24);
    layerButton.onClicked = function () {
      var dialogPane = new ht.widget.Dialog();
      var formPane = new ht.widget.FormPane();
      formPane.getView().style.background = '#090413';
      // 重写行背景
      // formPane.getRowBackground = function(row) {
      //   if (formPane.getRows().indexOf(row) === 0 || formPane.getRows().indexOf(row) === 4 ||
      //     formPane.getRows().indexOf(row) === 5 || formPane.getRows().indexOf(row) === 6 ||
      //     formPane.getRows().indexOf(row) === 8 || formPane.getRows().indexOf(row) === 12) {
      //     if (row.items[0].element.isSelected() === true) {
      //       return '#39B1E4'
      //     } else {
      //       return '#090413'
      //     }
      //   } else {
      //     return row ? row.background : null
      //   }
      // };
      // modify by shuai.zhao for bug 12228 【神东项目0125定制版本】手机端底图筛选页面较大无法进行滚动筛选 2021-06-17
      // 人员
      formPane.addRow([
        {
          id: '$button1',
          button: {
            label: '人       员',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'group_icon',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            onClicked: function () {
              let color = '';
              let selectFlag = false;
              if (formPane.getItemById('$button1').element.isSelected()) {
                color = '#39B1E4';
                selectFlag = true;
              } else {
                color = '#090413';
                selectFlag = false;
              }
              formPane.getRows()[0].background = color;
              for (let i = 4; i <= 6; i++) {
                formPane.getItemById('$button' + i).element.setSelected(selectFlag);
              }
            }
          }
        },
        null,
        null
      ], [0.3, 0.3, 0.3], 30, { background: '#090413' });
      formPane.addRow([
        {
          id: '$button4',
          button: {
            label: '领导',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/领导.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('领导') === false ? false : true
          }
        },
        {
          id: '$button5',
          button: {
            label: '员工',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/员工.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('员工') === false ? false : true
          }
        },
        {
          id: '$button6',
          button: {
            label: '外委队',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/外委队.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('外委队') === false ? false : true
          }
        },
      ], [0.3, 0.3, 0.3], 60);

      // 车辆
      formPane.addRow([
        {
          id: '$button7',
          button: {
            label: '车       辆',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'group_icon',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            onClicked: function () {
              let color = '';
              let selectFlag = false;
              if (formPane.getItemById('$button7').element.isSelected()) {
                color = '#39B1E4';
                selectFlag = true
              } else {
                color = '#090413';
                selectFlag = false
              }
              formPane.getRows()[0].background = color;
              for (let i = 10; i <= 17; i++) {
                formPane.getItemById('$button' + i).element.setSelected(selectFlag);
              }
            }
          }
        },
        null,
        null
      ], [0.3, 0.3, 0.3], 30, { background: '#090413' });
      formPane.addRow([
        {
          id: '$button10',
          button: {
            label: '皮卡指挥车',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/皮卡指挥车.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('皮卡指挥车') === false ? false : true
          }
        },
        {
          id: '$button11',
          button: {
            label: '材料车',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/材料车.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('材料车') === false ? false : true
          }
        },
        {
          id: '$button12',
          button: {
            label: '工程车',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/工程车.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('工程车') === false ? false : true
          }
        },
      ], [0.3, 0.3, 0.3], 60);
      formPane.addRow([
        {
          id: '$button13',
          button: {
            label: '运人车',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/运人车.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('运人车') === false ? false : true
          }
        },
        {
          id: '$button14',
          button: {
            label: '指挥车',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/指挥车.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('指挥车') === false ? false : true
          }
        },
        {
          id: '$button15',
          button: {
            label: '特种车',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/特种车.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('特种车') === false ? false : true
          }
        },
      ], [0.3, 0.3, 0.3], 60);
      formPane.addRow([
        {
          id: '$button16',
          button: {
            label: '防暴车',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/防暴车.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('防暴车') === false ? false : true
          }
        },
        {
          id: '$button17',
          button: {
            label: '低污染车',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/低污染车.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('低污染车') === false ? false : true
          }
        },
        null
      ], [0.3, 0.3, 0.3], 60);

      // 基站
      formPane.addRow([
        {
          id: '$button19',
          button: {
            label: '基       站',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'group_icon',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            onClicked: function () {
              let color = '';
              let selectFlag = false;
              if (formPane.getItemById('$button19').element.isSelected()) {
                color = '#39B1E4';
                selectFlag = true
              } else {
                color = '#090413';
                selectFlag = false
              }
              formPane.getRows()[0].background = color;
              for (let i = 22; i <= 22; i++) {
                formPane.getItemById('$button' + i).element.setSelected(selectFlag);
              }
            }
          }
        },
        null,
        null
      ], [0.3, 0.3, 0.3], 30, { background: '#090413' });
      formPane.addRow([
        {
          id: '$button22',
          button: {
            label: '基站',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/基站.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('基站') === false ? false : true
          }
        },
        null,
        null
      ], [0.3, 0.3, 0.3], 60);

      // 地测点
      formPane.addRow([
        {
          id: '$button23',
          button: {
            label: '地  测  点',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'group_icon',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            onClicked: function () {
              let color = '';
              let selectFlag = false;
              if (formPane.getItemById('$button23').element.isSelected()) {
                color = '#39B1E4';
                selectFlag = true
              } else {
                color = '#090413';
                selectFlag = false
              }
              formPane.getRows()[0].background = color;
              for (let i = 24; i <= 24; i++) {
                formPane.getItemById('$button' + i).element.setSelected(selectFlag);
              }
            }
          }
        },
        null,
        null
      ], [0.3, 0.3, 0.3], 30, { background: '#090413' });
      formPane.addRow([
        {
          id: '$button24',
          button: {
            label: '地测点',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/地测点.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('地测点') === false ? false : true
          }
        },
        null,
        null
      ], [0.3, 0.3, 0.3], 60);

      // 安全测点
      formPane.addRow([
        {
          id: '$button25',
          button: {
            label: '安全测点',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'group_icon',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            onClicked: function () {
              let color = '';
              let selectFlag = false;
              if (formPane.getItemById('$button25').element.isSelected()) {
                color = '#39B1E4';
                selectFlag = true
              } else {
                color = '#090413';
                selectFlag = false
              }
              formPane.getRows()[0].background = color;
              for (let i = 28; i <= 37; i++) {
                formPane.getItemById('$button' + i).element.setSelected(selectFlag);
              }
            }
          }
        },
        null,
        null
      ], [0.3, 0.3, 0.3], 30, { background: '#090413' });
      formPane.addRow([
        {
          id: '$button28',
          button: {
            label: '瓦斯',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/瓦斯.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('瓦斯') === false ? false : true
          }
        },
        {
          id: '$button29',
          button: {
            label: '烟雾',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/烟雾.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('烟雾') === false ? false : true
          }
        },
        {
          id: '$button30',
          button: {
            label: '温度',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/温度.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('温度') === false ? false : true
          }
        },
      ], [0.3, 0.3, 0.3], 60);
      formPane.addRow([
        {
          id: '$button31',
          button: {
            label: '一氧化碳',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/一氧化碳.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('一氧化碳') === false ? false : true
          }
        },
        {
          id: '$button32',
          button: {
            label: '二氧化碳',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/二氧化碳.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('二氧化碳') === false ? false : true
          }
        },
        {
          id: '$button33',
          button: {
            label: '风速',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/风速.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('风速') === false ? false : true
          }
        },
      ], [0.3, 0.3, 0.3], 60);
      formPane.addRow([
        {
          id: '$button34',
          button: {
            label: '压力',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/压力.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('压力') === false ? false : true
          }
        },
        {
          id: '$button35',
          button: {
            label: '水位',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/水位.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('水位') === false ? false : true
          }
        },
        {
          id: '$button36',
          button: {
            label: '综合测点',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/综合测点.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('综合测点') === false ? false : true
          }
        },

      ], [0.3, 0.3, 0.3], 60);
      formPane.addRow([
        {
          id: '$button37',
          button: {
            label: '安全其他',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/安全其他.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('安全其他') === false ? false : true
          }
        },
        null, null
      ], [0.3, 0.3, 0.3], 60);

      // 生产设备
      formPane.addRow([
        {
          id: '$button44',
          button: {
            label: '生产设备',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'group_icon',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            onClicked: function () {
              let color = '';
              let selectFlag = false;
              if (formPane.getItemById('$button44').element.isSelected()) {
                color = '#39B1E4';
                selectFlag = true
              } else {
                color = '#090413';
                selectFlag = false
              }
              formPane.getRows()[0].background = color;
              for (let i = 45; i <= 62; i++) {
                formPane.getItemById('$button' + i).element.setSelected(selectFlag);
              }
            }
          }
        },
        null,
        null
      ], [0.3, 0.3, 0.3], 30, { background: '#090413' });
      formPane.addRow([
        {
          id: '$button45',
          button: {
            label: '皮带电机',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/皮带电机.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('皮带电机') === false ? false : true
          }
        },
        {
          id: '$button46',
          button: {
            label: '风门',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/风门.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('风门') === false ? false : true
          }
        },
        {
          id: '$button47',
          button: {
            label: '风筒',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/风筒.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('风筒') === false ? false : true
          }
        },

      ], [0.3, 0.3, 0.3], 60);
      formPane.addRow([
        {
          id: '$button48',
          button: {
            label: '排水泵',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/排水泵.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('排水泵') === false ? false : true
          }
        },
        {
          id: '$button49',
          button: {
            label: '采掘设备',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/采掘设备.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('采掘设备') === false ? false : true
          }
        },
        {
          id: '$button50',
          button: {
            label: '工作面',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/工作面.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('工作面') === false ? false : true
          }
        },

      ], [0.3, 0.3, 0.3], 60);
      formPane.addRow([
        {
          id: '$button51',
          button: {
            label: '采煤机',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/采煤机.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('采煤机') === false ? false : true
          }
        },
        {
          id: '$button52',
          button: {
            label: '转载机',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/转载机.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('转载机') === false ? false : true
          }
        },
        {
          id: '$button53',
          button: {
            label: '破碎机',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/破碎机.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('破碎机') === false ? false : true
          }
        },

      ], [0.3, 0.3, 0.3], 60);
      formPane.addRow([
        {
          id: '$button54',
          button: {
            label: '电机',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/电机.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('电机') === false ? false : true
          }
        },
        {
          id: '$button55',
          button: {
            label: '变电所',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/变电所.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('变电所') === false ? false : true
          }
        },
        {
          id: '$button56',
          button: {
            label: '掘锚机',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/掘锚机.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('掘锚机') === false ? false : true
          }
        },

      ], [0.3, 0.3, 0.3], 60);
      formPane.addRow([
        {
          id: '$button57',
          button: {
            label: '连运机',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/连运机.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('连运机') === false ? false : true
          }
        },
        {
          id: '$button58',
          button: {
            label: '梭车',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/梭车.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('梭车') === false ? false : true
          }
        },
        {
          id: '$button59',
          button: {
            label: '锚杆机',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/锚杆机.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('锚杆机') === false ? false : true
          }
        },

      ], [0.3, 0.3, 0.3], 60);
      formPane.addRow([
        {
          id: '$button60',
          button: {
            label: '水泵房',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/水泵房.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('水泵房') === false ? false : true
          }
        },
        {
          id: '$button61',
          button: {
            label: '离心泵',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/离心泵.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('离心泵') === false ? false : true
          }
        },
        {
          id: '$button62',
          button: {
            label: '生产其他',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/生产其他.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('生产其他') === false ? false : true
          }
        },
      ], [0.3, 0.3, 0.3], 60);
      // 摄像头
      formPane.addRow([
        {
          id: '$button63',
          button: {
            label: '摄像头',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/摄像头.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            onClicked: function () {
              let color = '';
              let selectFlag = false;
              if (formPane.getItemById('$button63').element.isSelected()) {
                color = '#39B1E4';
                selectFlag = true
              } else {
                color = '#090413';
                selectFlag = false
              }
              formPane.getRows()[0].background = color;
              for (let i = 64; i <= 64; i++) {
                formPane.getItemById('$button' + i).element.setSelected(selectFlag);
              }
            }
          }
        },
        null,
        null
      ], [0.3, 0.3, 0.3], 30, { background: '#090413' });
      formPane.addRow([
        {
          id: '$button64',
          button: {
            label: '摄像头',
            labelColor: 'white',
            borderColor: '#090413',
            icon: 'images/摄像头.png',
            togglable: true,
            orientation: 'h',
            background: '#090413',
            selected: dm.a('摄像头') === false ? false : true
          }
        },
        null,
        null
      ], [0.3, 0.3, 0.3], 60);
      // end
      var buttons = [{
        label: '取消',
        action: function () {
          dialogPane.hide();
        }
      }, {
        label: '确定',
        action: function () {
          for (let j = 1; j <= 64; j++) {
            if (j !== 1 && j !== 7 && j !== 19 && j !== 25 && j !== 44 && j !== 63) {
              if (formPane.getItemById('$button' + j)) {
                let isSelect = formPane.getItemById('$button' + j).element.isSelected();
                let text = formPane.getItemById('$button' + j).element.getLabel().replace(/\s/g, '');
                dm.a(text, isSelect);
              }
            }
          }
          graphView.dm().each(node => {
            // 只需控制覆盖物, 其余节点不需处理
            if (node.getImage && node.getImage().indexOf('symbols') !== -1) {
              let url = node.getImage();
              let belongLayer = ht.Default.getImage(url).belongLayer;
              if (node.a('belongLayer') && dm.a(node.a('belongLayer').split('-')[1]) === false) {
                node.s('2d.visible', false);
              } else if (node.a('belongLayer') === undefined && belongLayer && dm.a(belongLayer.split('-')[1]) === false) {
                node.s('2d.visible', false);
              } else {
                node.s('2d.visible', true);
              }
            }
          })
          dialogPane.hide();
        }
      }];
      dialogPane.setConfig({
        title: '图层选择',
        draggable: true,
        width: 300,
        height: 420,
        position: {
          x: window.innerWidth - 300,
          y: 0
        },
        contentPadding: 4,
        content: formPane,
        buttons: buttons,
        buttonsAlign: 'right'
      });
      dialogPane.show();
    }
    layerButton.getView().style.top = 0;
    layerButton.getView().style.right = 0;
    layerButton.getView().style.width = 50;
    layerButton.getView().style.height = 30;
    graphView.getView().appendChild(layerButton.getView());

    // 创建定时器执行图纸SQL基站信息查询语句, 派发变量值改变
    var statement = graphView.getDataModel().a('sqlStatement');
    if (statement) {
      var temSQLInterval = setInterval(function () {
        exeKPSQLExecute(statement);
      }, 2000);
    }
  }, 200)
  });

  function zoomFunc(event) {

  }

  function throttle(func, delay) {
    var prev = Date.now();
    var timer = null;
    var args = arguments;
    return function (event) {
      var context = this;
      var now = Date.now();
      var remaining = delay - (now - prev);
      clearTimeout(timer)
      if (remaining <= 0) {
        func.apply(context, arguments);
        prev = Date.now();
      } else {
        timer = setTimeout(func, remaining)
      }
      arguments[0] = null;
    }
  }

  function debounce(func, delay) {
    let timeout;
    return function () {
      let context = this;
      let args = arguments;
      if (timeout) {
        clearTimeout(timeout);
      }
      timeout = setTimeout(() => {
        func.apply(this, args);
      }, delay)
    }
  }

  // add by jingyu.liu shuai.zhao  for bug 11876 【神东项目0125定制版本】手机端端底图缩放时卡顿，不够流畅 2021-05-28
  var canvas = graphView.getView().childNodes[0];
  var ctx = canvas.getContext('2d');
  var canvaswidth = canvas.width;
  var canvasheight = canvas.height;
  setTimeout(function () {
    canvaswidth = canvas.width;
    canvasheight = canvas.height;
  }, 500);

  var img = null;
  var zoom = 0;
  var isPinch = false;
  setTimeout(function () {
    Promise.all([
      createImageBitmap(canvas, 0, 0, canvas.clientWidth * KM_PixelRatio, canvas.clientHeight * KM_PixelRatio)
    ]).then(function (sprites) {
      img = sprites[0];
      imgzoom = 1;
      isPinch = true;
    })
  }, 2500)

  var zoomin = 0.97;
  var zoomout = 1.03;
  window.imgzoom = 1;
  var touchX = 0;
  var touchY = 0;
  var centerPointG;
  var zoomG = null;
  function resetHandlePinch() {
    graphView.onPinchEnded = function () {
      graphView.getDataModel().enableAnimation();
      graphView.setZoom(zoom, false, centerPointG)
      var timer2 = setTimeout(function () {
        Promise.all([
          createImageBitmap(canvas, 0, 0, canvas.clientWidth * KM_PixelRatio, canvas.clientHeight * KM_PixelRatio)
        ]).then(function (sprites) {
          img = sprites[0];
          imgzoom = 1;
        })
        zoomFunc(zoomG);
        clearTimeout(timer2);
        window.isonup = true;
      }, 200);
    }

    graphView.handlePinch = function (centerPoint, dist, lastDist) {
      window.isonup = false;
      try {
        if (isFast) {
          return false;
        }
        graphView.getDataModel().disableAnimation();

        if (zoom === 0) {
          zoom = graphView.getZoom();
        }

        // modify by shuai.zhao for bug 12174 【神东项目0125定制版本】巷道图操作过快时不跟手 2021-06-17
        // modify by shuai.zhao for bug 12650 【神东项目0125定制版本】底图缩小放大进行一定的倍数限制 2021-07-08
        if (dist > lastDist) {
          // zoomout = (dist / lastDist > 2) ? 2 : dist / lastDist
          if (zoom * zoomout <= 6) {
            zoom = zoom * zoomout;
            imgzoom = imgzoom * zoomout;
          }
        } else {
          // zoomin = (dist / lastDist < 0.5) ? 0.5 : dist / lastDist
          if (zoom * zoomin >= 0.04) {
            zoom = zoom * zoomin;
            imgzoom = imgzoom * zoomin;
          }
        }
        // end
        // end

        centerPointG = centerPoint;
        // modify by shuai.zhao for bug 12991 【神东项目0125定制版本】移动端反复进入退出智能感知，页面报错 2021-07-16
        if (img) {
          ctx.clearRect(0, 0, canvas.width, canvas.height);
          ctx.drawImage(img, (touchX * (1 - imgzoom) / 2) * KM_PixelRatio, (touchY * (1 - imgzoom) / 2) * KM_PixelRatio, canvas.width * imgzoom, canvas.height * imgzoom);
        }
        // end
      } catch (error) {
        alert(error)
      }
      // end
    }

    graphView.addInteractorListener(function (event) {
      event.event.stopPropagation();
      event.event.preventDefault();
      switch (event.kind) {
        case "betweenPinch": {
          touchX = event.event.touches[0].clientX + event.event.touches[1].clientX - 2 * canvas.clientLeft;
          touchY = event.event.touches[0].clientY + event.event.touches[1].clientY - 2 * canvas.clientTop;
        }
          break;
        case "endPan": {
          Promise.all([
            createImageBitmap(canvas, 0, 0, canvas.clientWidth * KM_PixelRatio, canvas.clientHeight * KM_PixelRatio)
          ]).then(function (sprites) {
            img = sprites[0];
            imgzoom = 1;
          })
        }
          break;
        default: {
          break;
        }
      }
      return false;
    })
  }
  resetHandlePinch()
  // end

  // 添加图纸放缩回调监听
  graphView.addPropertyChangeListener(debounce(function (event) {
    zoomG = event;
    if (event && event.property == 'zoom' && window.beginZoomRange !== undefined&& imgzoom ==1) {
      zoomFunc(event)
      setTimeout(function(){
        Promise.all([
          createImageBitmap(canvas, 0, 0, canvas.clientWidth * KM_PixelRatio, canvas.clientHeight * KM_PixelRatio)
        ]).then(function (sprites) {
          img = sprites[0];
        })
      },400)
    }
  }, 140))

  // 添加导航轨迹显示时,禁用图元的点击事件
  // graphView.setSelectableFunc(function (data) {
  //   if (imgzoom === 1) {
  //     let isNodeEnable = graphView.dm().getDatas().toArray().some(item => {
  //       return item.getClassName() === 'ht.Edge' && item.getStyle && item.getStyle('2d.visible') === true
  //     })
  //     if (isNodeEnable) {
  //       graphView.sm().cs();
  //       return false
  //     } else {
  //       return true
  //     }
  //   }
  // })

  // 动画管理Map
  window.animationMap = new Map();

  // 动态创建管理Map
  window.dynamicCreationMap = new Map();

  // 图扑获取外层KP window对象回调管理Map
  window.KPBodyCb = null;

  // 储存全局的节点图层可见信息
  window.showNodeArr = [];

  // 图扑获取外层KP window对象
  window.getKPBody = function (cb) {
    var postData = {
      type: 'getKPBody'
    };
    window.parent.postMessage(postData, '*');
    window.KPBodyCb = cb;
  }

  // 图扑通过组件名称获取相关属性
  window.getKPWindowAttr = function (attr) {
    let result = '';
    switch (attr) {
      case 'width':
        result = window.KPBody.getElementsByClassName('run-frame')[0].style.width
        break;
      case 'height':
        result = window.KPBody.getElementsByClassName('run-frame')[0].style.height
        break;
      case 'left':
        result = window.KPBody.getElementsByClassName('run-frame')[0].style.left
        break;
      case 'top':
        result = window.KPBody.getElementsByClassName('run-frame')[0].style.top
        break;
      default:
        break;
    }
    return result
  }

  // 图扑通过组件名称获取相关属性
  window.getKPObjectAttrByName = function (objectName, attr) {
    function searchChild(target, id) {
      let _child = null;
      for (let i = 0; i < target.childNodes.length; i++) {
        if (target.childNodes[i].id === id) {
          _child = target.childNodes[i];
          break;
        } else if (target.childNodes[i].childNodes.length !== 0) {
          _child = searchChild(target.childNodes[i], id)
          if (_child !== null) {
            return _child
          }
        }
      }
      return _child
    }
    if (window.KPBody !== undefined && window.KPObjects !== undefined) {
      let objectID = '';
      Object.keys(window.KPObjects).forEach(key => {
        if (key === objectName) {
          objectID = window.KPObjects[key];
        }
      })
      if (objectID === '') {
        alert('未找到该名称组件, 请检查名称!')
      } else {
        let targetEle = searchChild(window.KPBody, objectID);
        if (targetEle !== null) {
          return targetEle.style[attr];
        } else {
          alert('未查找到目标组件, 请检查!');
          return ''
        }
      }
    } else {
      alert('请先调用getKPBody方法获取KP组件');
      return
    }
  }
  // 自定义全局方法-图扑调用KP setPage方法
  window.setKPPage = function (pageName, layoutName, region = 'center', showType = 1) {
    if (!pageName || typeof pageName !== 'string') {
      return 'failed, pageName 入参不正确, 请检查';
    } else if (!layoutName || typeof layoutName !== 'string') {
      return 'failed, layoutName 入参不正确, 请检查';
    } else {
      var postData = {
        type: 'setKPPage',
        pageName: pageName,
        layoutName: layoutName,
        region: region,
        showType: showType
      };
      window.parent.postMessage(postData, '*');
      return 'success'
    }
  }

  // 自定义全局方法-图扑调用KP 动态创建模型对象方法
  window.KPCreateGraphicObject = function (pageName, modelType, modelName, objectName, left, top, width, height, zOrder, visible) {
    if (pageName === undefined || modelType === undefined || modelName === undefined || objectName === undefined || left === undefined ||
      top === undefined || width === undefined || height === undefined || zOrder === undefined || visible === undefined) {
      alert('参数不全,请检查!');
      return
    }
    let postData = {
      type: 'KPCreateGraphicObject',
      pageName: pageName,
      args: [modelType, modelName, objectName, left, top, width, height, zOrder, visible]
    };
    window.parent.postMessage(postData, '*');
  }

  // 自定义全局方法-图扑调用KP 布局内动态创建模型对象方法
  window.KPCreateGraphicObjectByLayout = function (pageName, modelType, modelName, objectName, containerName, zOrder, visible) {
    if (pageName === undefined || modelType === undefined || modelName === undefined || objectName === undefined ||
      containerName === undefined || zOrder === undefined || visible === undefined) {
      alert('参数不全,请检查!');
      return
    }
    let postData = {
      type: 'KPCreateGraphicObjectByLayout',
      pageName: pageName,
      args: [modelType, modelName, objectName, containerName, zOrder, visible]
    };
    window.parent.postMessage(postData, '*');
  }

  // 自定义全局方法-图扑调用KP 2D工艺流程图内动态创建模型对象方法
  window.KPCreateGraphicObjectByTwoDimensionalLayout = function (pageName, modelType, modelName, objectName, containerName, left, top, width, height, zOrder, visible) {
    if (pageName === undefined || modelType === undefined || modelName === undefined || objectName === undefined || containerName === undefined ||
      left === undefined || top === undefined || width === undefined || height === undefined || zOrder === undefined || visible === undefined) {
      alert('参数不全,请检查!');
      return
    }
    let postData = {
      type: 'KPCreateGraphicObjectByTwoDimensionalLayout',
      pageName: pageName,
      args: [modelType, modelName, objectName, containerName, left, top, width, height, zOrder, visible]
    };
    window.parent.postMessage(postData, '*');
  }

  // 自定义全局方法-图扑调用KP 动态删除模型对象方法
  window.KPDeleteGraphicObjectByName = function (pageName, modelName, graphicObjectName, cb) {
    if (pageName === undefined || modelName === undefined || graphicObjectName === undefined) {
      alert('参数不全,请检查!');
      return
    }
    let postData = {
      type: 'KPDeleteGraphicObjectByName',
      pageName: pageName,
      args: [modelName, graphicObjectName]
    };
    window.parent.postMessage(postData, '*');
    window.dynamicCreationMap.set(JSON.stringify(postData), cb);
  }

  // 自定义全局方法-图扑调用KP 获取动态创建模型对象方法
  window.KPGetGraphicObjectByName = function (pageName, modelName, graphicObjectName, cb) {
    if (pageName === undefined || modelName === undefined || graphicObjectName === undefined) {
      alert('参数不全,请检查!');
      return
    }
    let postData = {
      type: 'KPGetGraphicObjectByName',
      pageName: pageName,
      args: [modelName, graphicObjectName]
    };
    window.parent.postMessage(postData, '*');
    window.dynamicCreationMap.set(JSON.stringify(postData), cb);
  }

  // 自定义全局方法-图扑调用KP 获取动态创建模型对象方法
  window.KPGetGraphicObjectAttributeByName = function (pageName, modelName, objectName, attributeName, cb) {
    if (pageName === undefined || modelName === undefined || objectName === undefined || attributeName === undefined) {
      alert('参数不全,请检查!');
      return
    }
    let postData = {
      type: 'KPGetGraphicObjectAttributeByName',
      pageName: pageName,
      args: [modelName, objectName, attributeName]
    };
    window.parent.postMessage(postData, '*');
    window.dynamicCreationMap.set(JSON.stringify(postData), cb);
  }

  // 自定义全局方法-图扑调用KP 获取动态创建模型对象方法
  window.KPSetGraphicObjectAttributeByName = function (pageName, modelName, objectName, attributeName, value, cb) {
    if (pageName === undefined || modelName === undefined || objectName === undefined || attributeName === undefined || value === undefined) {
      alert('参数不全,请检查!');
      return
    }
    let postData = {
      type: 'KPSetGraphicObjectAttributeByName',
      pageName: pageName,
      args: [modelName, objectName, attributeName, value]
    };
    window.parent.postMessage(postData, '*');
    window.dynamicCreationMap.set(JSON.stringify(postData), cb);
  }

  // 自定义全局方法-图扑设置当前选中节点名,供KP调用使用
  window.setCurrentNodeName = function (displayName) {
    if (!displayName || typeof displayName !== 'string') {
      return 'failed, displayName 入参不正确, 请检查'
    }
    var postData = {
      type: 'setCurrentNodeName',
      displayName: displayName
    };
    window.parent.postMessage(postData, '*');
    return 'success'
  }

  // 自定义全局方法-图扑调用KP SQLExecute语句
  window.exeKPSQLExecute = function (statement) {
    var postData = {
      type: 'exeKPSQLExecute',
      statement: statement
    };
    window.parent.postMessage(postData, '*');
  }

  // 自定义全局方法-图扑主动更新当前图级至KP
  window.updateCurrentZoomToKP = function (zoom) {
    var postData = {
      type: 'updateCurrentZoomToKP',
      zoom: zoom
    };
    window.parent.postMessage(postData, '*');
  }
  var handleEvent = function (e) {
    coor = graphView.getLogicalPoint(e)
    var postData = {
      type: 'getDrawCoordinate',
      data: {
        coordinate: graphView.getLogicalPoint(e)
      }
    };
    window.parent.postMessage(postData, "*");
  }

  // 实现iframe和父页面的通讯
  window.addEventListener('message', function (e) {
    // 监听KP 调用图扑方法
    if (imgzoom === 1) {
      if (typeof e.data === 'object') {
        var postData = e.data;
        var method = postData.method;
        function transFuncToString(obj) {
          for (let i = 0; i < obj.d.length; i++) {
            let nodeStyle = obj.d[i].s;
            for (let key in nodeStyle) {
              if (typeof nodeStyle[key] === 'function') {
                nodeStyle[key] = nodeStyle[key].toString();
              }
            }
          }
        }
        // KP 获取图扑属性
        if (postData.type === 'get') {
          var node = null;
          graphView.getDataModel().each(function (Node1) {
            if (Node1.getDisplayName() === postData.nodeName) {
              node = Node1;
            }
          });
          try {
            node[method](...postData.args);
          } catch (err) {
            console.log(err);
          }
          // KP 设置图扑属性
        } else if (postData.type === 'set') {
          var node = null;
          graphView.getDataModel().each(function (Node1) {
            if (Node1.getDisplayName() === postData.nodeName) {
              node = Node1;
            }
          });
          try {
            node[method](...postData.args);
          } catch (err) {
            console.log(err);
          }
          // KP 调用图扑动画
        } else if (postData.type === 'setAnimation') {
          graphView.getDataModel().each(node => {
            if (node.getDisplayName() === postData.targetName) {
              try {
                // 动画对象中function转换处理
                let animationObj = postData.animationObj;
                for (let key in animationObj) {
                  if (animationObj[key].easing && animationObj[key].easing.indexOf('function') !== -1) {
                    let newFunc = new Function('return ' + animationObj[key].easing);
                    animationObj[key].easing = newFunc();
                  }
                  if (animationObj[key].onUpdate && animationObj[key].onUpdate.indexOf('function') !== -1) {
                    let newFunc = new Function('return ' + animationObj[key].onUpdate);
                    animationObj[key].onUpdate = newFunc();
                  }
                  if (animationObj[key].onComplete && animationObj[key].onComplete.indexOf('function') !== -1) {
                    let newFunc = new Function('return ' + animationObj[key].onComplete);
                    animationObj[key].onComplete = newFunc();
                  }
                }
                node.setAnimation(animationObj);
              } catch (err) {
                console.log(err);
              }
            }
          });
          // KP 调用图扑动画 暂停/继续 功能
        } else if (postData.type === 'toggleAnimation') {
          graphView.getDataModel().each(node => {
            if (node.getDisplayName() === postData.targetName) {
              if (node._pauseAnimation) {
                node.resumeAnimation();
              } else {
                node.pauseAnimation();
              }
            }
          });
          // KP 调用图扑轨迹导航展示
        } else if (postData.type === 'showTrack') {
          var trackArr = postData.track;
          trackArr.forEach((item, i) => {
            let beginNodeName = item.beginInfo.beginNodeName;
            let endNodeName = item.endInfo.endNodeName;
            let trackPath = globalGraph.getShortestPath(beginNodeName, endNodeName);
            if (trackPath === -1) {
              console.log('导航路径有误,请检查!');
              return
            }
            let pathArr = trackPath.split('->');
            graphView.getDataModel().each(node => {
              if (node.getClassName() === 'ht.Edge') {
                let tmpIndex1 = pathArr.indexOf(node.getSource().getDisplayName());
                let tmpIndex2 = pathArr.indexOf(node.getTarget().getDisplayName());
                if (tmpIndex1 !== -1 && tmpIndex2 !== -1) {
                  if (Math.abs(tmpIndex1 - tmpIndex2) === 1) {
                    node.s('2d.visible', postData.isShow);
                  }
                }
              }
            });
          });
          // KP 调用图扑轨迹回放
        } else if (postData.type === 'trackPlayback') {
          // 开始动画具体内容函数
          function beginMove(node, points, segments, offset) {
            let position = ht.Default.getPercentPositionOnPoints(points, segments, offset);
            if (position.x !== undefined && position.y !== undefined) {
              node.setX(position.x);
              node.setY(position.y);
            }
          }
          // 动画对象
          var anim = null;
          // 目标节点
          var targetNode = null;
          // 轨迹数组
          var trackArr = postData.track;
          // 获取轨迹所有关键点集合
          function trackArrLine(trackArr) {
            graphView.getDataModel().each(node => {
              if (node.getDisplayName() === postData.target) {
                targetNode = node;
              }
              if (node.getClassName() === 'ht.Edge') {
                trackArr.forEach((item, i) => {
                  let beginNodeName = item.beginInfo.beginNodeName;
                  let endNodeName = item.endInfo.endNodeName;
                  // 如果起止点与绘图连线顺序一致
                  if (node.getSource().getDisplayName() === beginNodeName && node.getTarget().getDisplayName() === endNodeName) {
                    let points = [];
                    points.push({
                      x: node.getSource().getPosition().x,
                      y: node.getSource().getPosition().y
                    });
                    let middlePoints = node.s('edge.points') ? node.s('edge.points')._as : [];
                    points = points.concat(middlePoints);
                    points.push({
                      x: node.getTarget().getPosition().x,
                      y: node.getTarget().getPosition().y
                    });
                    trackArr[i].line = points;
                    // 显示轨迹
                    node.s('2d.visible', true);
                    // 如果起止点与绘图连线顺序相反
                  } else if (node.getSource().getDisplayName() === endNodeName && node.getTarget().getDisplayName() === beginNodeName) {
                    let points = [];
                    points.push({
                      x: node.getTarget().getPosition().x,
                      y: node.getTarget().getPosition().y
                    });
                    let middlePoints = node.s('edge.points') ? [...node.s('edge.points')._as].reverse() : [];
                    points = points.concat(middlePoints);
                    points.push({
                      x: node.getSource().getPosition().x,
                      y: node.getSource().getPosition().y
                    });
                    trackArr[i].line = points;
                    // 显示轨迹
                    node.s('2d.visible', true);
                  }

                });

              }
            });
          }
          trackArrLine(trackArr)
          if (targetNode !== null) {
            // 所有原始点
            var allPoints = [];
            // 动画总耗时
            var allTimeArr = [];
            // add by jie.li 2021-6-30 bug 12478 【神东项目0125定制版本】轨迹回放函数trackPlayback 能够和导航一样，根据底图已有的线，给不相邻的两个基站找到一条路线
            for (var i = 0; i < trackArr.length; i++) {
              let item = trackArr[i];
              if (item.line) {
                allPoints = allPoints.concat(item.line);
              } else {
                // let sourceNode = null;
                // let targetNode = null;
                // graphView.dm().each(node => {
                //   if (node.getDisplayName() === item.beginInfo.beginNodeName) {
                //     sourceNode = node;
                //   }
                //   if (node.getDisplayName() === item.endInfo.endNodeName) {
                //     targetNode = node;
                //   }
                // })
                // if (sourceNode !== null && targetNode !== null) {
                //   allPoints.push(sourceNode.getPosition());
                //   allPoints.push(targetNode.getPosition());
                // }
                //   // 重新规划轨迹最佳路线
                let beginNodeName = item.beginInfo.beginNodeName;
                let endNodeName = item.endInfo.endNodeName;
                let tracPlaykPath = globalGraph.getShortestPath(beginNodeName, endNodeName);
                if (tracPlaykPath === -1) {
                  console.log('导航路径有误,请检查!');
                  return
                }
                let PlayPathArr = tracPlaykPath.split('->');
                let newItem = [];
                for (let key = 0; key < PlayPathArr.length - 1; key++) {
                  let item2 = {
                    beginInfo: {
                      beginNodeName: '22煤1200米候车点FZ181',
                      arriveTime: '2020-10-10 14:00:00',
                      leaveTime: '2020-10-10 14:30:00'
                    },
                    endInfo: {
                      endNodeName: '辅运平硐400米FZ193',
                      arriveTime: '2020-10-10 15:00:00',
                      leaveTime: '2020-10-10 15:30:00'
                    }
                  };
                  item2.beginInfo.beginNodeName = PlayPathArr[key];
                  item2.endInfo.endNodeName = PlayPathArr[key + 1];
                  newItem.push(item2)
                }
                trackArrLine(newItem)
                this.console.log(newItem)
                trackArr.splice(i, 1)
                trackArr.splice(i, 0, ...newItem)
                i = i - 1;

              }
              // end
              let seconds = new Date(item.endInfo.arriveTime).getTime() - new Date(item.beginInfo.leaveTime).getTime()
              allTimeArr.push(seconds);
            }
            // 去重, 排除停留点重复项
            var newAllPoints = allPoints.filter((item, i) => {
              return i === 0 || (i > 0 && (item.x !== allPoints[i - 1].x || item.y !== allPoints[i - 1].y));
            });
            // 设置默认路径轨迹连接方式为直线连接
            var segments = new Array(newAllPoints.length).fill(2);
            segments[0] = 1;
            // 动画对象ID
            var animID = postData.animID;
            // 计算每段线耗时的百分比
            var TimePercent = new Array(trackArr.length);
            // 总时间秒数
            var durationTime = eval(allTimeArr.join('+'));
            allTimeArr.forEach((time, i) => {
              TimePercent[i] = time / durationTime;
            });
            // 区间判断函数
            function judgeSection(value) {
              // return value < 0.66 ? 0.66 * value : 1.2 * value
              for (let i = 0; i < TimePercent.length; i++) {
                if (TimePercent[i] > value) {
                  return TimePercent[i];
                }
              }
              return TimePercent[TimePercent.length - 1]
            }
            if (window.animationMap.has(animID)) {
              let oldAnim = window.animationMap.get(postData.animID);
              window.animationMap.delete(animID)
              oldAnim.stop(false);
            }
            graphView.setSelectableFunc(() => { return false })
            // 启动动画
            anim = ht.Default.startAnim({
              duration: durationTime / 360,
              easing: function (t) {
                return t
                //  return judgeSection(t);
              },
              action: function (v, t) {
                beginMove(targetNode, newAllPoints, segments, v * 100)
              },
              finishFunc: function () {
                if (window.animationMap.has(animID)) {
                  var postData = {
                    type: 'trackPlaybackFinish',
                    status: 'finished',
                    animID: animID
                  };
                  graphView.setSelectableFunc(() => { return true })
                  window.parent.postMessage(postData, '*');
                }
              }
            });
            // 动画对象管理
            window.animationMap.set(postData.animID, anim);
          }
          // KP 调用图扑轨迹回放暂停/继续 切换功能
        } else if (postData.type === 'togglePlayback') {
          let anim = window.animationMap.get(postData.animID);
          if (anim.isPaused()) {
            anim.resume();
          } else {
            anim.pause();
          }
          // KP 调用图扑轨迹回放倍速功能
        } else if (postData.type === 'changePlaybackSpeed') {
          let anim = window.animationMap.get(postData.animID);
          anim.duration = anim.duration / postData.speed;
          // KP 调用图扑动态创建图标
        }

        else if (postData.type === 'createSymbol') {
          var url = 'symbols/' + postData.symbolName + '.json';
          var objectArr = postData.objectArr;
          // added by shangwei.tian for bug{动态创建图标，自定义属性获取异常} at 2021/05/28
          var orginalProperty = ['displayName', 'position', 'anchor', 'isFollowScale', 'width', 'height'];
          // end by shangwei.tian at 2021/05/28
          objectArr.forEach((item, i) => {
            let node = new ht.Node();
            // 缩放卡顿问题
            var obj = {
              "2d.editable": false,
              "2d.movable": false,
              'pixelPerfect': true,
              'preventDefaultWhenInteractive': false
            }
            node.s(obj);
            // 缩放卡顿问题
            node.setImage(url);
            var belongLayer = '';
            if (ht.Default.getImage(url)) {
              belongLayer = ht.Default.getImage(url).belongLayer;
            }
            if (belongLayer) {
              node.a('belongLayer', belongLayer)
              node.s('2d.visible', graphView.dm().a(belongLayer.split('-')[1]));
            }
            if (item.displayName !== undefined) {
              node.setDisplayName(item.displayName);
            }
            if (item.position !== undefined) {
              node.setPosition(item.position.x, item.position.y);
            }
            if (item.anchor !== undefined) {
              node.setAnchor(item.anchor.x, item.anchor.y);
            }
            if (item.isFollowScale !== undefined) {
              node.a('isFollowScale', item.isFollowScale);
            } else {
              node.a('isFollowScale', false);
            }
            if (item.width !== undefined) {
              node.setWidth(item.width);
            }
            if (item.height !== undefined) {
              node.setHeight(item.height);
            }
            // added by shangwei.tian for bug{动态创建图标，自定义属性获取异常} at 2021/05/28
            for (var key in item) {
              if (!orginalProperty.includes(key)) {
                node.a(key, item[key]);
              }
            }
            // end by shangwei.tian at 2021/05/28
            // 进行默认缩放
            if (window.beginZoomRange) {
              let zoomRate = window.beginZoomRange / graphView.getZoom();
              if (zoomRate !== 1 && node.setScale) {
                // modify by shuai.zhao for bug 12651 【神东项目0125定制版本】底图放大过程中，人员车辆等一直在缩小，最终缩小到不能点击 2021-07-16
                if ((1 / zoomRate) <= 16) {
                  node.setScale(zoomRate, zoomRate);
                } else {
                  node.setScale(1 / 16, 1 / 16);
                }
                // end
              }
            }
            graphView.getDataModel().add(node);
          });
          // KP 调用图扑动态删除图标
        }
         // 根据所给坐标进行轨迹回放
         else if(postData.type ==='RoutePlayBack'){
          function beginMove(node, points, segments, offset) {
            let position = ht.Default.getPercentPositionOnPoints(points, segments, offset);
            if (position.x !== undefined && position.y !== undefined) {
              node.setX(position.x);
              node.setY(position.y);
            }
          }
          var linedata = postData.palette
          var edge = [];
          if (linedata && linedata.length>1) {
            for (let index = 0; index < linedata.length; index++) {
              var shape = new ht.Shape();
              graphView.dm().add(shape);
              shape.setStyle("shape.background",null)
              shape.setStyle("shape.border.width",linedata[index].width)
              shape.setStyle("shape.border.color",linedata[index].color)
              shape.setPoints(linedata[index].position)
              edge.push(linedata[index].position[0])
              edge.push(linedata[index].position[1])

            }
            // 设置默认路径轨迹连接方式为直线连接
          var segments = new Array(edge.length).fill(2);
          segments[0] = 1;
            graphView.getDataModel().each(node => {
              if (node.getDisplayName() === postData.node) {
                targetNode = node;
                targetNode.setPosition(linedata[0].position[0].x,linedata[0].position[0].y)
              }
            })
            // 启动动画
            anim = ht.Default.startAnim({
              duration: 10000,
              easing: function (t) {
                return t
              },
              action: function (v, t) {
                beginMove(targetNode, edge, segments, v * 100)
              },
              // finishFunc: function () {
              //   alert("到达终点")
              // }
            });
          }
        }
         //根据坐标点以及颜色值来动态创建多个多边形以及自定义点击事件
         else if (postData.type === 'rectangularPolygon') {
          var polygonShape = new CreateShapeInteractor(graphView);
          var polygons = postData.data;
          if(polygons){
            for (let index = 0; index < polygons.length; index++) {
              var newPolygon = polygonShape.draw_shape(polygons[index].Polygon)
              newPolygon.setStyle('shape.background', polygons[index].PolygonBackground);
              newPolygon.setName(polygons[index].name)
              newPolygon.s({'label.font':postData.data[index].label,
                "label.color":postData.data[index].PolygonBackground
              })
              newPolygon.s('2d.movable', false);
              newPolygon.a("clickColor",postData.data[index].clickColor)
              newPolygon.a('PolygonBackground',postData.data[index].PolygonBackground)
              newPolygon.s("onClick",function(event, data, view) {
                setCurrentNodeName(data._name)
                if (graphView.dm()) {
                  graphView.dm().each(node => {
                    if(node.getName()&& data.getName() && node.getName() ===  data.getName() && data.getAttrObject().clickColor){
                      node.setStyle('shape.background', node.getAttrObject().clickColor);
                      node.setStyle('shape.border.color', node.getAttrObject().clickColor);
                    }else if(node.getName()&& data.getName() && node.getName() !==  data.getName() && data.getAttrObject().PolygonBackground){
                      if(node.getAttrObject().PolygonBackground){
                        node.setStyle('shape.background', node.getAttrObject().PolygonBackground);
                        node.setStyle('shape.border.color', node.getAttrObject().PolygonBackground);
                      }else if(node.getAttrObject().rectColor){
                        node.setStyle('shape.background', node.getAttrObject().rectColor);
                        node.setStyle('shape.border.color', node.getAttrObject().rectColor);
                      }

                    }
                  })
                }
                var newpostData = {
                  type: 'PolygonCallback',
                  name:data.getName()
                };
                window.parent.postMessage(newpostData, '*');
              })
              newPolygon.s("interactive",true)
            }
          }
        }
        else if (postData.type === 'createSymbolApp') {
          var url = 'symbols/' + postData.symbolName + '.json';
          var objectArr = postData.objectArr;
          // added by shangwei.tian for bug{动态创建图标，自定义属性获取异常} at 2021/05/28
          var orginalProperty = ['displayName', 'position', 'anchor', 'isFollowScale', 'width', 'height'];
          // end by shangwei.tian at 2021/05/28
          objectArr.forEach((item, i) => {
            let node = new ht.Node();
            // 缩放卡顿问题
            var obj = {
              "2d.editable": false,
              "2d.movable": false,
              'pixelPerfect': true,
              'preventDefaultWhenInteractive': false
            }
            node.s(obj);
            // 缩放卡顿问题
            node.setImage(url);
            var belongLayer = '';
            if (ht.Default.getImage(url)) {
              belongLayer = ht.Default.getImage(url).belongLayer;
            }
            if (belongLayer) {
              node.a('belongLayer', belongLayer)
              node.s('2d.visible', graphView.dm().a(belongLayer.split('-')[1]));
            }
            if (item.displayName !== undefined) {
              node.setDisplayName(item.displayName);
            }
            if (item.position !== undefined) {
              node.setPosition(item.position.x, item.position.y);
            }
            if (item.anchor !== undefined) {
              node.setAnchor(item.anchor.x, item.anchor.y);
            }
            if (item.isFollowScale !== undefined) {
              node.a('isFollowScale', item.isFollowScale);
            } else {
              node.a('isFollowScale', false);
            }
            if (item.width !== undefined) {
              node.setWidth(item.width);
            }
            if (item.height !== undefined) {
              node.setHeight(item.height);
            }
            // added by shangwei.tian for bug{动态创建图标，自定义属性获取异常} at 2021/05/28
            for (var key in item) {
              if (!orginalProperty.includes(key)) {
                node.a(key, item[key]);
              }
            }
            // end by shangwei.tian at 2021/05/28
            // 进行默认缩放
            if (window.beginZoomRange) {
              let zoomRate = window.beginZoomRange / graphView.getZoom();
              if (zoomRate !== 1 && node.setScale) {
                // modify by shuai.zhao for bug 12651 【神东项目0125定制版本】底图放大过程中，人员车辆等一直在缩小，最终缩小到不能点击 2021-07-16
                if ((1 / zoomRate) <= 16) {
                  node.setScale(zoomRate, zoomRate);
                } else {
                  node.setScale(1 / 16, 1 / 16);
                }
                // end
              }
            }
            node.s("onClick",(event,data,view)=>{
              var nodedata = {
                type:"createcallback",
                name:data.getDisplayName()
              };
              this.window.parent.postMessage(nodedata,"*")
            })
            graphView.getDataModel().add(node);
          });
          // KP 调用图扑动态删除图标
        }

        else if (postData.type === 'deleteSymbol') {
          var nameArr = postData.nameArr;
          graphView.getDataModel().getDatas().toArray().forEach(node => {
            if (nameArr.indexOf(node.getDisplayName()) !== -1) {
              graphView.getDataModel().remove(node);
            }
          })
          // KP 调用获取图扑所有数据
        } else if (postData.type === 'getAllDatas') {
          // 节点对象中有任何的function则不可通过postMessage传递, 进行递归处理为字符串
          var jsonObj = graphView.getDataModel().toJSON();
          transFuncToString(jsonObj);
          var postData = {
            type: 'getAllDatas',
            data: jsonObj
          };
          window.parent.postMessage(postData, '*');
          // KP 调用获取图扑指定数据
        } else if (postData.type === 'getData') {
          // 节点对象中有任何的function则不可通过postMessage传递, 进行递归处理为字符串
          var data = null;
          let oldJson = graphView.dm().toJSON();
          let dataModel = new ht.DataModel();
          dataModel.deserialize(oldJson);
          var name = postData.displayName;
          dataModel.getDatas().toArray().forEach(item => {
            if (item.getDisplayName() === name) {
              data = item
            } else {
              dataModel.remove(item);
            }
          })
          var jsonObj = dataModel.toJSON();
          transFuncToString(jsonObj);
          var postData = {
            type: 'getData',
            data: jsonObj,
            displayName: name
          };
          window.parent.postMessage(postData, '*');
          // KP 调用图扑缩放到全部
        } else if (postData.type === 'zoomToAll') {
          graphView.fitContent()
          window.zoomimageall = true;
          // KP 调用图扑缩放到目标对象
        } else if (postData.type === 'zoomToTarget') {
          var name = postData.target;
          var currentZoom = graphView.getZoom();
          console.log(currentZoom);
          var data = null;
          graphView.dm().each(item => {
            if (item.getDisplayName() === name) {
              data = item;
            }
          })
          graphView.fitData(data);
          graphView.setZoom(postData.zoom);
          // 监听KP 发送查询sql数据至HT
        } else if (postData.type === 'sendSQLDataToHT') {
          if (postData.data) {
            window[postData.dataName] = null;
            window[postData.dataName] = postData.data;
            // 数据派发至节点
            if (window.baseStationInfo) {
              window.baseStationInfo.forEach(item => {
                var baseStationName = item.name;
                var baseStationEmp = item.empcount || 0;
                var baseStationCar = item.carcount || 0;
                var baseStationID = item.number;
                graphView.dm().each(node => {
                  var bindingVariable = node.a('bindingVariable');
                  if (bindingVariable) {
                    for (let key in bindingVariable) {
                      if (bindingVariable[key].split('@')[0] === baseStationName) {
                        let bindPropertyName = bindingVariable[key].split('@')[1];
                        node.a('number', baseStationID);
                        node.a(key, bindPropertyName === '人员数' ? baseStationEmp : baseStationCar)
                        node.iv();
                      }
                    }
                  }
                })
              })
            }
          }
          // 监听图扑调用KP DeleteGraphicObjectByName返回
        } else if (postData.type === 'KPDeleteGraphicObjectByName') {
          let cb = window.dynamicCreationMap.get(JSON.stringify(postData.originKey));
          if (cb !== undefined && cb !== null) {
            cb(postData.result);
          }
          // 监听图扑调用KP GetGraphicObjectByName返回
        } else if (postData.type === 'KPGetGraphicObjectByName') {
          let cb = window.dynamicCreationMap.get(JSON.stringify(postData.originKey));
          if (cb !== undefined && cb !== null) {
            cb(postData.result);
          }
          // 监听图扑调用KP GetGraphicObjectAttributeByName返回
        } else if (postData.type === 'KPGetGraphicObjectAttributeByName') {
          let cb = window.dynamicCreationMap.get(JSON.stringify(postData.originKey));
          if (cb !== undefined && cb !== null) {
            cb(postData.result);
          }
          // 监听图扑调用KP SetGraphicObjectAttributeByName返回
        } else if (postData.type === 'KPSetGraphicObjectAttributeByName') {
          let cb = window.dynamicCreationMap.get(JSON.stringify(postData.originKey));
          if (cb !== undefined && cb !== null) {
            cb(postData.result);
          }
          // 监听图扑获取外层KP window对象返回
        } else if (postData.type === 'getKPBody') {
          function parseVNode(vnode) {
            let type = vnode.type;
            let _node = null;
            if (type === 1) {
              let data = vnode.data;
              let tag = vnode.tag;
              let children = vnode.children;
              _node = document.createElement(tag);
              Object.keys(data).forEach(key => {
                let attrName = key;
                let attrValue = data[key];
                _node.setAttribute(attrName, attrValue);
              })
              children.forEach(child => {
                if (child !== null) {
                  _node.appendChild(parseVNode(child));
                }
              })
            } else if (type === 3) {
              return document.createTextNode(vnode.value);
            }
            return _node
          }
          window.KPBody = parseVNode(postData.data)
          window.KPObjects = postData.objects;
          window.KPBodyCb(true);
          // 监听KP 操作图扑鹰眼功能
        } else if (postData.type === 'operateHTOverview') {
          let operateType = postData.operateType;
          switch (operateType) {
            case 'open':
              document.getElementById('htoverview').style.display = 'block';
              break;
            case 'close':
              document.getElementById('htoverview').style.display = 'none';
              break;
            case 'expand':
              if (document.getElementById('htoverview').style.display !== 'none') {
                overview._expand = false;
                document.getElementById('overviewButton').click();
              }
              break;
            case 'collapse':
              if (document.getElementById('htoverview').style.display !== 'none') {
                overview._expand = true;
                document.getElementById('overviewButton').click();
              }
              break;
            default:
              break;
          }
        } else if (postData.type === 'drawDomain') {
          graphView.setInteractors([
            new CreateRectInteractor(graphView)
          ]);
        } else if (postData.type === 'getDrawCoordinate') {
          graphView.getView().addEventListener('click', handleEvent);
        } else if (postData.type === 'removeGetDrawCoordinate') {
          graphView.getView().removeEventListener('click', handleEvent);
        } else if (postData.type === 'getDrawDomainDataByCoordinate') {
          var node = new ht.Node();
          var coordinate = postData.data;
          node.setSize(Math.abs(coordinate[3].x - coordinate[0].x), Math.abs(coordinate[3].y - coordinate[0].y));
          node.setPosition((coordinate[3].x + coordinate[0].x) / 2, (coordinate[3].y + coordinate[0].y) / 2);
          node.setStyle('shape.border.width.absolute', true);
          node.setStyle('shape', 'rect');
          node.setStyle('border.color', '#1c4adf');
          node.setStyle('shape.border.width', 2);
          node.setStyle('shape.background', 'rgba(115,115,115,0)');
          node.s('2d.movable', false);
          node.setName('closeRect');
          graphView.dm().add(node);
          if (postData.zoom) {
            var datadom = ''
            graphView.dm().each(item => {
              if (item.getName() === "closeRect") {
                datadom = item;
              }
            })
            graphView.fitData(datadom)
            graphView.setZoom(postData.zoom);
          }
          var rect = {
            x: coordinate[0].x,
            y: coordinate[0].y,
            width: coordinate[3].x - coordinate[0].x,
            height: coordinate[3].y - coordinate[0].y
          }
          // 框选区域内所有图元
          timeout = setTimeout(() => {
            var modeName = graphView.getDatasInRect(rect, false)._as;
            var modeListAll = {};
            modeName.forEach(node => {
              if (node.getImage && node.getImage().indexOf('symbols') !== -1 && node.a('belongLayer')) {
                modeListAll[node._displayName] = node.getAttrObject().belongLayer
              }
            });
            var postData = {
              type: 'getDrawDomainDataByCoordinate',
              data: {
                modeNameList: modeListAll
              }
            };
            window.parent.postMessage(postData, '*');
          }, 500)

          // 清除显示的边框
        } else if (postData.type === 'closeDrawDomain') {
          let dataModel = graphView.getDataModel();
          dataModel.getDatas().toArray().forEach(item => {
            if (item.getName() === postData.data) {
              graphView.dm().remove(item)
            }
          })
        } else if (postData.type === 'drawPolygon') {
          graphView.setInteractors([
            new CreateShapeInteractor(graphView)
          ]);
        } else if (postData.type === 'getDrawPolygonDataByCoordinate') {
          var polygonShape = new CreateShapeInteractor(graphView)
          polygonShape.draw_shape(postData.data)
          if (postData.zoom) {
            var datadom = ''
            graphView.dm().each(item => {
              if (item.getName() === "closePolygon") {
                datadom = item;
              }
            })
            graphView.fitData(datadom)
            graphView.setZoom(postData.zoom);
            window.newPostData = postData.data._as;
            window.newGraphView = graphView;
            timeout = setTimeout(() => {
              var domeAll = polygonShape.polygon_node(window.newPostData, window.newGraphView)
              var postData = {
                type: 'getDrawPolygonDataByCoordinate',
                data: {
                  modeNameList: domeAll
                }
              };
              // 返回获取的信息
              window.parent.postMessage(postData, '*');
            }, 500)
          }
          // 根据所属图层来隐藏图元
        } else if (postData.type === 'shieldCovering') {
          var belongLayerAll = postData.belongLayer;
          var isDisplay = postData.isDisplay
          if (graphView.dm()) {
            graphView.dm().each(node => {
              if (node.getImage && node.getImage().indexOf('symbols') !== -1) {
                let url = node.getImage();
                var belongLayer = ht.Default.getImage(url).belongLayer;
                if (belongLayer && belongLayer.indexOf('-') !== -1) {
                  var belongLayerMax = belongLayer.split('-')[0];
                  var belongLayerMin = belongLayer.split('-')[1];
                  for (const key in belongLayerAll) {
                    var belongLayerArr = belongLayerAll[key];
                    if (belongLayerArr) {
                      if (belongLayerArr === "All" && belongLayerMax === key) {
                        graphView.dm().getAttrObject()[belongLayerMin] = isDisplay;
                        node.s('2d.visible', isDisplay);
                      } else {
                        for (const index in belongLayerArr) {
                          if (belongLayerArr[index] === belongLayerMin) {
                            graphView.dm().getAttrObject()[belongLayerArr[index]] = isDisplay;
                            node.s('2d.visible', isDisplay);
                          }
                        }
                      }
                    }
                  }
                }
              }
            })
          }
        }
      }
    }
  })
  // add by shuai.zhao for bug 13243 【神东项目0125定制版本】底图缩放时，如果轻微快速地滑动，底图基本不能缩放 2021-08-04
  var touchStart = [], touchEnd = [], startTime, endTime;
  document.getElementsByTagName('div')[0].addEventListener('touchstart', function (e) {
    startTime = new Date().getTime()
    if (e.touches.length >= 2) {
      touchStart = e.touches
    }

  })
  document.getElementsByTagName('div')[0].addEventListener('touchmove', function (e) {
    if (e.touches.length >= 2) {
      touchEnd = e.touches
    }
  })
  document.getElementsByTagName('div')[0].addEventListener('touchend', function (e) {
    endTime = new Date().getTime()
    if (endTime - startTime <= 100 && touchStart.length !== 0 && touchEnd.length !== 0) {
      isFast = true;
      var startDistance = Math.sqrt((touchStart[0].pageX - touchStart[1].pageX) ** 2 + (touchStart[0].pageY - touchStart[1].pageY) ** 2)
      var endDistance = Math.sqrt((touchEnd[0].pageX - touchEnd[1].pageX) ** 2 + (touchEnd[0].pageY - touchEnd[1].pageY) ** 2)

      var screenDistance = Math.sqrt(window.screen.width ** 2 + window.screen.height ** 2)
      if (startDistance - endDistance < 0) {
        0 === zoom && (zoom = graphView.getZoom());
        var num = 0;
        var outNum = ((endDistance - startDistance) / screenDistance * 6) / 5 + 1;
        var timerOut = setInterval(() => {
          num++;
          if ((zoom * outNum) <= 6.0) {
            zoom *= outNum;
            imgzoom *= outNum
          }
          ctx.clearRect(0, 0, canvas.width, canvas.height),
            ctx.drawImage(img, touchX * (1 - imgzoom) / 2 * KM_PixelRatio, touchY * (1 - imgzoom) / 2 * KM_PixelRatio, canvas.width * imgzoom, canvas.height * imgzoom)
          if (num === 5) {
            graphView.setZoom(zoom);
            Promise.all([createImageBitmap(canvas, 0, 0, canvas.clientWidth * KM_PixelRatio, canvas.clientHeight * KM_PixelRatio)]).then(function (e) {
              img = e[0], imgzoom = 1
            });
            zoomFunc(zoomG)
            isFast = false;
            clearInterval(timerOut)
          }
        }, 20);
      } else {
        0 === zoom && (zoom = graphView.getZoom());
        var num = 0;
        var inNum = ((endDistance - startDistance) / screenDistance * 6) / 5 + 1;
        console.log(inNum, zoom)
        var timerIn = setInterval(() => {
          num++;
          console.log(zoom * inNum, num)
          if ((zoom * inNum) >= 0.04) {
            zoom *= inNum;
            imgzoom *= inNum
          }
          ctx.clearRect(0, 0, canvas.width, canvas.height),
            ctx.drawImage(img, touchX * (1 - imgzoom) / 2 * KM_PixelRatio, touchY * (1 - imgzoom) / 2 * KM_PixelRatio, canvas.width * imgzoom, canvas.height * imgzoom)
          if (num === 5) {
            graphView.setZoom(zoom);
            Promise.all([createImageBitmap(canvas, 0, 0, canvas.clientWidth * KM_PixelRatio, canvas.clientHeight * KM_PixelRatio)]).then(function (e) {
              img = e[0], imgzoom = 1
            });
            zoomFunc(zoomG)
            isFast = false;
            clearInterval(timerIn)
          }
        }, 20);

      }
    }

    touchStart = [];
    touchEnd = [];


  })
}
