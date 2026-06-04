import logging

from or_config import config

logger = logging.getLogger(__name__)

if config("EXP_METRICS", cast=bool, default=False):
    logger.info(">>> Using experimental metrics")
else:
    pass