import logging

from or_config import config

logger = logging.getLogger(__name__)
if config("EXP_METRICS", cast=bool, default=False):
    logger.info(">>> Using experimental product-analytics")
    from .product_analytics_ch import *
else:
    from .product_analytics import *
