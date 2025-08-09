#!/bin/sh
cd /vod
cvdl all a02dc370efd2befeac97881dc83f11bb >> /var/log/cvdl-cron.log 2>&1
