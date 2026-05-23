/**
 * 页面级样式注入器。
 *
 * 页面样式来自 page.config.styleConfig，编辑态与预览态共用同一段 CSS。
 * 这里使用 render 函数创建 style 标签，避免在 SFC 模板中直接书写动态 style 标签。
 */
import { computed, defineComponent, h } from 'vue'
import { elevateStyleConfigPriority, replaceStyleConfigPlaceholders } from './style-config-css'

export default defineComponent({
  name: 'PageStyleInjector',
  props: {
    css: {
      type: String,
      default: '',
    },
    pageId: {
      type: String,
      default: '',
    },
  },
  setup(props) {
    const cssText = computed(() =>
      elevateStyleConfigPriority(
        replaceStyleConfigPlaceholders(props.css, {
          pageId: props.pageId,
        }),
      ),
    )

    return () => {
      if (!cssText.value) return null
      return h(
        'style',
        {
          type: 'text/css',
          'data-designer-page-style': props.pageId || 'current',
        },
        cssText.value,
      )
    }
  },
})
