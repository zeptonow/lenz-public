import logging

from . import smtp

logger = logging.getLogger(__name__)
# Log level is configured by app.py, using INFO as default here
logging.basicConfig(level=logging.INFO)

if smtp.has_smtp():
    logger.info("valid SMTP configuration found")
else:
    logger.info("no SMTP configuration found or SMTP validation failed")
