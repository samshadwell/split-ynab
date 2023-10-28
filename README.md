# split-ynab

Integration between YNAB and Splitwise.

## Overview

My partner and I use Splitwise for shared expenses, and I use YNAB to track my spending. We follow the "Option Two" as
outlined in [YNAB's blog post](https://support.ynab.com/en_us/splitwise-and-ynab-a-guide-H1GwOyuCq#register) on the
subject. Specifically:

1. When I make a purchase, the full dollar amount is recorded in YNAB as a split transaction: (usually) half the
   transaction comes from a normal category, and the remainder comes from a "Splitting" category.
2. When a settle-up transaction happens it's a split transaction consisting of:
   1. Purchases my parter made on my behalf, categorized as outflows in the relevant categories
   2. The sum of their part of all purchases I made on their behalf, categorized as an inflow in the "Splitting"
      category

This integration makes performing the above bookkeeping easier.

### "I Pay" Transactions

When I make a purchase I log it in YNAB as a simple (non-split) transaction, with the correct category for my portion
of the split. For example, if we eat at a restaurant and the bill is $50, I create a $50 transaction coming from my
"Restaurant" budget. I also give the transaction a specific tag.

Then, this integration picks up the tagged transaction and:

1. Converts it into a split transaction. In the example above, it would split the transaction into a $25 "Restaurant"
   transaction (my portion), and a $25 "Splitting" transaction (their portion).
2. Adds the transaction to Splitwise.
3. Removes the tag from the transaction.

### "Settle Up" Transactions

When a settle-up transaction occurs in Splitwise, the integration will create a transaction in YNAB. This transaction
will be pre-split based on the transactions it represents, though none of the splits have categories they do preserve
their memos from Splitwise. I must then manually categorize each of the splits, but it does save me the math!

## Configuration

TODO
