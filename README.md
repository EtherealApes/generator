# Generator
Goals of the generator
- Produce images composed of multiple PNG's with weighted attributes and custom rulesets
- Produce metadata that reflects the images, ready for IPFS

## How to
- Prepare your attributes list
- Add them to the nft struct as individual slice of strings variables
- Add them to the metadata struct as string variables
- Populate asset store and assign to in progress generated NFTs
- Write getter methods that define rarity and rulesets for combinations
- Add individual traits to the metadata (or keep them hidden if desired)
- Define draw order of assets (or rule based draw orders)
- Define Collection Title
- Define data directory for importing assets
- Define trait formatting (default title case) if necessary