"""
Configuration loader for GMGN monitor
Similar to cmd/script/config.go
"""

import os
from dotenv import load_dotenv


class Config:
    """Configuration for GMGN monitor"""

    def __init__(self):
        self.debug = False
        self.metrics_port = 2113
        self.chains = []

    @classmethod
    def load(cls):
        """Load configuration from environment"""
        # Load .env file if it exists
        load_dotenv()

        config = cls()

        # Load configuration from environment variables
        config.debug = os.getenv('DEBUG', 'true').lower() == 'true'
        config.metrics_port = int(os.getenv('METRICS_PORT', '2113'))

        # Parse chains list
        chains_str = os.getenv('CHAINS', 'sol,bsc,base')
        config.chains = [c.strip() for c in chains_str.split(',') if c.strip()]

        return config

    def __repr__(self):
        return f"Config(debug={self.debug}, metrics_port={self.metrics_port}, chains={self.chains})"
