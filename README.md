# split-ynab

Integration between YNAB and Splitwise.

## Overview

My partner and I use a combination of a shared credit card account and Venmo for shared expenses, and I use YNAB to
track my spending. We roughly follow the "Option Two" as outlined in
[YNAB's blog post](https://support.ynab.com/en_us/splitwise-and-ynab-a-guide-H1GwOyuCq#register) on the subject.
This integration makes the bookkeeping easier. Our setup is:

1. We have a shared credit card account that is used for expenses which we split evenly (groceries, restaurants, etc.).
   This is an account in my name which my partner is an authorized user of.
1. When either of us makes a purchase on that account, it will be synced to my YNAB via the built-in import. I then
   classify the full amount of that purchase to the relevant category. So a $30 purchase at Trader Joe's will show up
   (initially) as coming entirely from my "Groceries" budget category.
1. This program is run on a periodic schedule and sees that a new expense has been added to the shared credit card
   account. It updates the transaction to be a split transaction. In this example, it will update the $30 purchase so
   that $15 come from my "Groceries" budget category, and $15 come from my "Splitting" category.
1. At the end of the month when we get the bill for the credit card, I pay the full amount and charge my partner for her
   portion via Venmo manually. When I get paid, that money goes into refilling the "Splitting" category.

## Configuration

Configuration is assumed to live in a `config.go` file at the root of the application. For security reasons, I don't
check this file into Git, but you can see the general structure in `config.go.example`. To use this yourself, first
run:

```shell
cp config.go.example config.go
```

Then fill in the values in `config.go` with your own values.
