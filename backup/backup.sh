#!/bin/bash
BACKUP_DIR=$HOME/Завантажене/tsubriy/devops_portfolio/backup
SOURCE_DIR=$HOME/Завантажене/tsubriy/devops_portfolio/source
LOG_FILE="$BACKUP_DIR/logfile.log"
log(){
    echo "[$(date '+%Y-%m-%d %H:%M:S')] $1" | tee -a "$LOG_FILE"
}
if [ ! -d "$SOURCE_DIR" ]; then
 log "ERROR: Directory $SOURCE_DIR doesn’t exist!"
 exit 1
fi
log "The start of process"
tar -cvf $BACKUP_DIR/backup-$(date +%Y%m%d-%H%M%S).tar.gz $SOURCE_DIR/TestDir/*
find $BACKUP_DIR -type f -name "*.tar.gz" | sort | head -n -5 | xargs -r rm
