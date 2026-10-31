<template>
  <span>
    <a class="button is-dark is-outlined is-small"
       :href="trackerUrl"
       target="_blank"
       @click.left.prevent="openDialog(item)"
       title="Search PornoLab">
      <span class="icon">
        <b-icon pack="mdi" icon="flask-outline" size="is-small" />
      </span>
    </a>
  </span>
</template>

<script>
export default {
  name: 'SearchPornolabButton',
  props: { item: { type: Object, default: () => ({}) } },
  computed: {
    trackerUrl () {
      if (!this.item || !this.item.title) return '#'
      let query = this.item.title
      if (this.item.site) {
        query = `${this.item.site} ${this.item.title}`
      }
      return `https://pornolab.net/forum/tracker.php?nm=${encodeURIComponent(query)}`
    }
  },
  methods: {
    openDialog (item) {
      if (!item) return
      this.$store.commit('overlay/showSearchPornolabScenes', { item })
    }
  }
}
</script>
