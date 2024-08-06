# sonarr-share

    services:
        sonarr-share:
            image: "ghcr.io/jleagle/sonarr-share:main"
            container_name: "sonarr-share"
            hostname: "sonarr-share"
            restart: "unless-stopped"
            entrypoint: "/root/sonarr-share -sonarr-key ${SONARR_KEY}"
            ports:
              - "7879:7879"
