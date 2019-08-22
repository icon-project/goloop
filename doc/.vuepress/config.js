module.exports = {
    base: '/doc/',
    title: 'Goloop Document',
    port: 10080,
    themeConfig: {
        sidebar: [
            ["/", "Home"],
            {
                title: 'Getting Started',
                children: [
                    '/build',
                    '/tutorial',
                ]
            },
            {
                title: 'Genesis',
                children: [
                    '/genesis_tx',
                    '/genesis_storage',
                ]
            },
            {
                title: 'JSON-RPC',   // required
                // path: '/jsonrpc',      // optional, which should be a absolute path.
                // collapsable: false, // optional, defaults to true
                sidebarDepth: 2,    // optional, defaults to 1
                children: [
                    '/jsonrpc_v3',
                    '/btp_extension',
                ]
            },
            {
                title: 'Management',
                children: [
                    '/goloop_admin_api',
                    ['/goloop_cli', "Goloop CLI"],
                    ['/metric', "Metric"],
                ]
            },
            //EndOfSidebar
        ],
        lastUpdated: 'Last Updated', // string | boolean
    },
}