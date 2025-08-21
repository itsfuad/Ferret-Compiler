package toml

type TOMLValue any

type TOMLTable map[string]TOMLValue

type TOMLData map[string]TOMLTable
