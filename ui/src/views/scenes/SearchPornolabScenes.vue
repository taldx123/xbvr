<template>
  <b-modal :active="isModalActive"
           has-modal-card
           trap-focus
           aria-role="dialog"
           @cancel="close"
           aria-modal
           can-cancel>
    <div class="modal-card">
      <header class="modal-card-head">
        <p class="modal-card-title">Search PornoLab: {{ scene.title }}</p>
        <a class="button is-small is-link is-outlined"
           style="margin-right: 0.75em;"
           :href="externalSearchUrl"
           target="_blank"
           rel="noreferrer"
           title="Open in new tab">
          <b-icon pack="mdi" icon="open-in-new" size="is-small" />
        </a>
        <button class="delete" @click="close" aria-label="close"></button>
      </header>
      <div class="modal-card-body">
        <b-field grouped>
          <b-input v-model="queryString"
                   placeholder="Search tracker..."
                   expanded
                   @keyup.native.enter="performSearch"
                   :loading="isFetching" />
          <p class="control">
            <b-button type="is-primary" :loading="isFetching" @click="performSearch">
              Search
            </b-button>
          </p>
        </b-field>

        <b-notification v-if="statusMessage" type="is-warning" :closable="false">
          {{ statusMessage }}
        </b-notification>

        <b-table :data="searchResults"
                 :loading="isFetching"
                 striped
                 hoverable
                 default-sort="date"
                 default-sort-direction="desc">
          <b-table-column field="category" label="Category" sortable v-slot="props" width="110">
            <small>{{ props.row.category }}</small>
          </b-table-column>

          <b-table-column field="title" label="Title" sortable v-slot="props">
            <a :href="props.row.topic_url" target="_blank" rel="noreferrer">
              <strong>{{ props.row.title }}</strong>
            </a>
          </b-table-column>

          <b-table-column field="size" label="Size" sortable numeric v-slot="props" width="105">
            <span class="no-wrap">{{ formatSize(props.row) }}</span>
          </b-table-column>

          <b-table-column field="seeders" label="S" sortable numeric v-slot="props" width="70">
            <span class="tag is-success is-light"><strong>{{ props.row.seeders }}</strong></span>
          </b-table-column>

          <b-table-column field="leechers" label="L" sortable numeric v-slot="props" width="70">
            <span class="tag is-light">{{ props.row.leechers }}</span>
          </b-table-column>

          <b-table-column field="date" label="Date" sortable v-slot="props" width="105">
            <small class="no-wrap">{{ props.row.date }}</small>
          </b-table-column>

          <b-table-column field="magnet" label="Magnet" v-slot="props" width="290">
            <div v-if="props.row.magnet" class="torrent_magnet" style="display:table-cell">
              <div class="torrent_magnet_inner" style="color:#cc0000">
                <svg aria-hidden="true" focusable="false" role="img" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" style="width: 1em; height: 1em; vertical-align: -0.125em; fill: currentColor;"><path d="M368 32h-64c-17.67 0-32 14.33-32 32v32h128V64c0-17.67-14.33-32-32-32zm-224 0H80C62.33 32 48 46.33 48 64v32h128V64c0-17.67-14.33-32-32-32zm128 96v128c0 35.35-28.65 64-64 64s-64-28.65-64-64V128H16v128c0 106.04 85.96 192 192 192s192-85.96 192-192V128H272z"/></svg><a style="padding-inline-start:0.3em"
                   :href="props.row.magnet"
                   title="Baixar via link magnético">{{ formatShortMagnet(props.row.magnet) }}</a><button
                   v-if="showFallbackDmm"
                   class="xbvr-dmm-fallback"
                   type="button"
                   @click.prevent="openInDmm(props.row.magnet)"
                   style="font-family: Roboto, sans-serif; margin-left: 5px; padding: 3px 5px; border-width: medium; border-style: none; border-color: currentcolor; border-image: none; border-radius: 3px; background: rgb(0, 160, 176); color: rgb(236, 240, 241); cursor: pointer; transition: background-color 0.3s, transform 0.1s; transform: scale(1);">DMM🧲</button>
              </div>
            </div>
            <b-button v-else
                      type="is-info"
                      size="is-small"
                      outlined
                      :loading="loadingMagnetId === props.row.id"
                      @click="resolveRowMagnet(props.row)"
                      title="Generate Magnet & DMM Link">
              <b-icon pack="mdi" icon="magnet" size="is-small" />
              <span>Magnet</span>
            </b-button>
          </b-table-column>

          <b-table-column label="DL" centered v-slot="props" width="60">
            <a class="button is-primary is-small"
               :href="getDownloadHref(props.row)"
               title="Download .torrent">
              <b-icon pack="mdi" icon="download" size="is-small" />
            </a>
          </b-table-column>

          <template #empty>
            <div class="has-text-centered" v-if="!isFetching">
              No results found.
            </div>
          </template>
        </b-table>
      </div>
    </div>
  </b-modal>
</template>

<script>
import ky from 'ky'
import prettyBytes from 'pretty-bytes'

export default {
  name: 'SearchPornolabScenes',
  data () {
    return {
      isModalActive: true,
      isFetching: false,
      loadingMagnetId: null,
      showFallbackDmm: true,
      unmounted: false,
      queryString: '',
      statusMessage: '',
      searchResults: [],
      scene: {}
    }
  },
  mounted () {
    const item = Object.assign({}, this.$store.state.overlay.searchPornolabScenes.scene)
    this.scene = item
    if (item && item.title) {
      this.queryString = item.site ? `${item.site} ${item.title}` : item.title
      this.performSearch()
    }
  },
  beforeDestroy () {
    this.unmounted = true
  },
  computed: {
    externalSearchUrl () {
      const q = this.queryString || (this.scene && this.scene.title) || ''
      return `https://pornolab.net/forum/tracker.php?nm=${encodeURIComponent(q)}`
    }
  },
  methods: {
    close () {
      this.$store.commit('overlay/hideSearchPornolabScenes')
    },
    performSearch () {
      const q = (this.queryString || '').trim()
      if (!q) return
      this.isFetching = true
      this.statusMessage = ''
      ky.get('/api/pornolab/search?q=' + encodeURIComponent(q), { timeout: 60000 })
        .json()
        .then(data => {
          this.searchResults = data.results || []
          this.statusMessage = data.status || ''
          this.isFetching = false
          this.checkDmmExtensionInjection()
        })
        .catch(err => {
          this.statusMessage = err.message || 'Search request failed'
          this.isFetching = false
        })
    },
    checkDmmExtensionInjection () {
      this.$nextTick(() => {
        setTimeout(() => {
          if (this.unmounted || !this.$el) return
          const extBtn = this.$el.querySelector('.torrent_magnet button:not(.xbvr-dmm-fallback)')
          this.showFallbackDmm = !extBtn
        }, 250)
      })
    },
    async resolveRowMagnet (row, silent = false) {
      if (!row || row.magnet) return
      this.loadingMagnetId = row.id
      const params = new URLSearchParams()
      if (row.download_url) params.set('url', row.download_url)
      if (row.id) params.set('id', row.id)
      if (row.title) params.set('title', row.title)

      try {
        const resp = await fetch(`/api/pornolab/magnet?${params.toString()}`)
        if (!resp.ok) {
          const errText = await resp.text()
          throw new Error(errText || `HTTP ${resp.status}`)
        }
        const data = await resp.json()
        if (data && data.magnet) {
          this.$set(row, 'magnet', data.magnet)
          this.checkDmmExtensionInjection()
        }
      } catch (e) {
        if (!silent) {
          this.$buefy.toast.open({
            message: e.message || 'Failed to resolve magnet link',
            type: 'is-danger',
            duration: 5000
          })
        }
      } finally {
        if (this.loadingMagnetId === row.id) {
          this.loadingMagnetId = null
        }
      }
    },
    formatShortMagnet (magnet) {
      if (!magnet) return ''
      return magnet.length > 30 ? magnet.slice(0, 30) + '...' : magnet
    },
    openInDmm (magnet) {
      if (!magnet) return
      const url = `https://debridmediamanager.com/library?addMagnet=${encodeURIComponent(magnet)}`
      window.open(url, '_blank', 'noopener,noreferrer')
    },
    formatSize (row) {
      if (row.size && row.size > 0) {
        return prettyBytes(row.size)
      }
      return row.size_str || '-'
    },
    getDownloadHref (row) {
      const params = new URLSearchParams()
      if (row.download_url) params.set('url', row.download_url)
      if (row.id) params.set('id', row.id)
      return `/api/pornolab/download?${params.toString()}`
    }
  }
}
</script>

<style scoped>
.modal-card {
  width: 90vw;
  max-width: 1300px;
  height: 80vh;
  margin: 0 auto;
}
.no-wrap {
  white-space: nowrap;
}
.torrent_magnet_inner:has(button + button) .xbvr-dmm-fallback {
  display: none !important;
}
</style>
