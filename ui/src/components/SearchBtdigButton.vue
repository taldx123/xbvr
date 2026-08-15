<template>
    <span>
        <a class="button is-dark is-outlined is-small"
            :href="btdigUrl"
            target="_blank"
            @click.left.prevent="openDialog(item)"
            title="Search BTDIG">
            <span class="icon">
              <b-icon pack="mdi" icon="magnify" size="is-small" />
           </span>
        </a>
    </span>
</template>

<script>
export default {
  name: 'SearchBtdigButton',
  props: { item: { type: Object, default: () => ({}) } },
  computed: {
    btdigUrl () {
      if (!this.item || !this.item.title) return '#'
      let query = this.item.title
      if (this.item.site) {
        query = `${this.item.site} ${this.item.title}`
      }
      return `https://pt.btdig.com/search?order=0&q=${encodeURIComponent(query)}`
    }
  },
  methods: {
    openDialog (item) {
      if (!item) return
      this.$store.commit('overlay/showSearchBtdigScenes', { item })
    }
  }
}
</script>
