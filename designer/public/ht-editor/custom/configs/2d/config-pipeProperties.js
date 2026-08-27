(function() {
    window.hteditor_config = window.hteditor_config || {};
    window.hteditor_config.valueTypes = window.hteditor_config.valueTypes || {};
    var previousHandler = window.hteditor_config.onTitleCreating;

    if (window.ht && ht.Default && !ht.Default.getImage('induforge.pipe.direction')) {
        ht.Default.setImage('induforge.pipe.direction', {
            width: 16, height: 10,
            comps: [{ type: 'shape', points: [1, 1, 15, 5, 1, 9], segments: [1, 2, 2], closePath: true, background: '#E7FFF6' }]
        });
    }

    window.hteditor_config.valueTypes.PipeWidth = {
        type: 'number',
        min: 1,
        max: 64,
        step: 1
    };
    window.hteditor_config.valueTypes.PipeFlowSpeed = {
        type: 'number',
        min: 0.25,
        max: 20,
        step: 0.25
    };
    window.hteditor_config.valueTypes.PipeFlowMode = {
        type: 'enum',
        values: ['off', 'continuous', 'segment', 'particle'],
        i18nLabels: ['PipeFlowOff', 'PipeFlowContinuous', 'PipeFlowSegment', 'PipeFlowParticle']
    };
    window.hteditor_config.valueTypes.PipeFlowCount = {
        type: 'number', min: 1, max: 100, step: 1
    };

    function numberAttribute(data, name, fallback) {
        var value = Number(data.a(name));
        return isFinite(value) ? value : fallback;
    }

    function applyPipeStyles(data) {
        if (!data || data.a('induforge.path.type') !== 'pipe') return;
        var mode = data.a('induforge.pipe.flowMode') || 'continuous';
        var running = data.a('induforge.pipe.running') !== false;
        var reverse = data.a('induforge.pipe.reverse') === true;
        var speed = numberAttribute(data, 'induforge.pipe.speed', 2);
        var elementSize = numberAttribute(data, 'induforge.pipe.elementSize', 12);
        var spacing = numberAttribute(data, 'induforge.pipe.spacing', 36);
        var count = numberAttribute(data, 'induforge.pipe.count', 6);
        var segmented = mode === 'segment';
        var particles = mode === 'particle' || mode === 'continuous';

        data.s({
            'shape.dash': true,
            'shape.dash.pattern': segmented ? [16, 8] : [100000, 0],
            'shape.dash.flow': segmented && running,
            'shape.dash.flow.reverse': reverse,
            'shape.dash.flow.step': speed,
            'flow': particles && running,
            'flow.reverse': reverse,
            'flow.step': speed,
            'flow.count': mode === 'continuous' ? Math.max(1, Math.min(3, count)) : count,
            'flow.element.image': 'induforge.pipe.direction',
            'flow.element.count': count,
            'flow.element.min': elementSize,
            'flow.element.max': elementSize,
            'flow.element.space': spacing,
            'flow.element.autorotate': true
        });
    }

    window.InduForgePipe = Object.freeze({ apply: applyPipeStyles });

    window.hteditor_config.onTitleCreating = function(editor, params) {
        if (previousHandler) {
            previousHandler(editor, params);
        }
        if (!params || params.title !== 'TitleShapeBackground') {
            return;
        }

        var inspector = params.inspector;
        if (!inspector || inspector.type !== 'data' || inspector.__induforgePipePropertiesAdded) {
            return;
        }
        inspector.__induforgePipePropertiesAdded = true;
        addPipeProperties(inspector);
    };

    function isPipe(inspector) {
        var data = inspector && inspector.data;
        return data instanceof ht.Shape && data.a('induforge.path.type') === 'pipe';
    }

    function addPipeProperties(inspector) {
        var S = hteditor.getString;
        var title = inspector.addTitle('PipeSettings');
        title.visible = isPipe;

        addProperty(inspector, 'shape.border.color', S('PipeWallColor'), 'Color');
        addProperty(inspector, 'shape.border.width', S('PipeOuterWidth'), 'PipeWidth');
        addProperty(inspector, 'shape.dash.color', S('PipeMediumColor'), 'Color');
        addProperty(inspector, 'shape.dash.width', S('PipeInnerWidth'), 'PipeWidth');
        addAttribute(inspector, 'induforge.pipe.flowMode', S('PipeFlowMode'), 'PipeFlowMode');
        addAttribute(inspector, 'induforge.pipe.running', S('PipeFlowEnabled'), 'Boolean');
        addAttribute(inspector, 'induforge.pipe.reverse', S('PipeFlowReverse'), 'Boolean');
        addAttribute(inspector, 'induforge.pipe.speed', S('PipeFlowSpeed'), 'PipeFlowSpeed');
        addAttribute(inspector, 'induforge.pipe.elementSize', S('PipeElementSize'), 'PipeWidth');
        addAttribute(inspector, 'induforge.pipe.spacing', S('PipeElementSpacing'), 'PipeWidth');
        addAttribute(inspector, 'induforge.pipe.count', S('PipeElementCount'), 'PipeFlowCount');
    }

    function addProperty(inspector, property, name, valueType) {
        inspector.addCustomProperty({
            accessType: 's',
            property: property,
            name: name,
            valueType: valueType,
            extraInfo: {
                visible: isPipe
            }
        });
    }

    function addAttribute(inspector, property, name, valueType) {
        inspector.addCustomProperty({
            accessType: 'a', property: property, name: name, valueType: valueType,
            onValueChanged: function(data) { applyPipeStyles(data); },
            extraInfo: { visible: isPipe }
        });
    }
})();
