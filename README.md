# split-ynab

Automatically split transactions in YNAB.

## Overview

My partner and I use a combination of a personal credit cards, a shared credit card (via authorized users), and Venmo
for shared expenses. I use YNAB to track my spending (and they do not). We roughly follow the "Option Two" as outlined
in [YNAB's blog post](https://support.ynab.com/en_us/splitwise-and-ynab-a-guide-H1GwOyuCq#register) on the subject.
This integration makes the bookkeeping easier. Our setup is:

1. We have a shared credit card account that is used for expenses which we split evenly (groceries, restaurants, etc.).
   This is an account in my name which my partner is an authorized user of.
1. In my budget I have a "Splitting" category which I've funded with $1000. This is the amount that I'm willing to spend
   on shared expenses before my partner needs to pay me back.
1. When either of us makes a purchase on that account, it will be synced to my YNAB via the built-in import. I then
   classify the full amount of that purchase to the relevant category. So a $30 purchase at Trader Joe's will show up
   (initially) as coming entirely from my "Groceries" budget category.
1. This program is run on a periodic schedule and sees that a new expense has been added to the shared credit card
   account. It updates the transaction to be a split transaction. In this example, it will update the $30 purchase so
   that $15 come from my "Groceries" budget category, and $15 come from my "Splitting" category.
1. At the end of the month when we get the bill for the credit card, I pay the full amount and charge my partner for
   their portion via Venmo manually. The amount they owe me is $1000 minus the current amount in my "Splitting"
   category. When I get repaid, that money goes to refilling the "Splitting" category.

We like this setup because it reduces the number of Venmo transactions we make, and I like that this keeps my YNAB
accurate with my spending, even when I pay for shared things. I also like that it keeps YNAB as the single source of
truth for my spending, rather than using another app (like Splitwise) to track shared expenses.

## Configuration

Configuration is read from a `config.yml` file at the root of the application. An example config might look like:

```yaml
---
ynabToken: "your-token-here"
budgetId: "00000000-1111-2222-3333-444455556666"
splitCategoryId: "deadbeef-1111-2222-3333-444455556666"
accounts:
  - id: "01010101-1111-2222-3333-444455556666" # Defaults to 50/50
    exceptFlags: ["green"]
  - id: "02020202-1111-2222-3333-444455556666"
    exceptFlags: ["green"]
    defaultPercentTheirShare: 20
flags:
  - color: "orange" # Defaults to 50/50
  - color: "purple"
    percentTheirShare: 30
```

`ynabToken` is your YNAB API token. You can get generate one on the
[YNAB developer settings page](https://app.ynab.com/settings/developer).

`budgetId` is the ID of the budget you want to use. You can get this from the URL of the budget in the YNAB web app. For
example, if your budget URL is `https://app.ynab.com/00000000-1111-2222-3333-444455556666/budget`, then your budget ID
is `00000000-1111-2222-3333-444455556666`.

`splitCategoryId` is the ID of the category you want to assign the other person's portion of the split transactions to,
the "Splitting" category in the example above. Getting this is a little trickier than the above, you'll have to issue
an API call to [list your categories](https://api.ynab.com/v1#/Categories/getCategories) and find the one you want.
For example, you can use `curl` and `jq` to find the ID of the category named "Splitting" like so (replacing
`<your_budget_id>` and `<your_ynab_token>` with the values you obtained above):

```shell
curl --request GET \
  --url https://api.ynab.com/v1/budgets/<your_budget_id>/categories \
  --header 'Authorization: Bearer <your_ynab_token>' \
  | jq '.data.category_groups[].categories[] | select(.name=="Splitting")'
```

The `accounts` and `flags` sections are used to determine which transactions should be split, and how to split them.

To have an account's transactions be split by default, first obtain the account's ID from the
[YNAB API](https://api.ynab.com/v1#/Accounts/getAccounts), then add an entry to the `accounts` section. You can
optionally specify `exceptFlags`: any transactions on the account with the given flags will _not_ be split. I use this
in cases where I don't want to split a transaction, like if I've bought something for myself on the shared credit card.
You can also optionally specify `defaultPercentTheirShare`: this is the percentage of the transaction that should be
considered the other person's responsibility, and defaults to 50. But if you have a different arrangement, like if you
pay 70% of the shared expenses and your partner pays 30%, you would specify `defaultPercentTheirShare: 30`.

`flags` is used for two things:

1. To allow transactions outside the specified `accounts` to be split
1. To override the default split percentage for a given transaction

For all transactions in your YNAB, if it matches one of these flags it will be split at the given rate, defaulting to
50/50. This is useful if pay for something on your personal credit card, but you want to split it with your partner.
It's also useful if you want to split a transaction at a different rate than the default for a given account, like if
you pay 70% of the internet bill but it comes out of the shared credit card account.
