module Sanitize exposing (date, parseDate, parseHM, parseNumber)

import Expect
import Parser exposing (run)
import Test exposing (..)
import UI.Sanitize as Sanitize


date : Test
date =
    describe "Test date sanitizer"
        [ test "2023-06-01" <|
            \_ -> Expect.equal (Sanitize.date "2023-06-01") "2023-06-01"
        , test "2023-066" <|
            \_ -> Expect.equal (Sanitize.date "2023-066") "2023-06"
        , test "2023-6-1" <|
            \_ -> Expect.equal (Sanitize.date "2023-6-01") "2023-06-01"
        , test "3000" <|
            \_ -> Expect.equal (Sanitize.date "3000") ""
        , test "2300" <|
            \_ -> Expect.equal (Sanitize.date "2300") "2"
        , test "2023-61" <|
            \_ -> Expect.equal (Sanitize.date "2023-61") "2023-6"
        , test "2023-12-31" <|
            \_ -> Expect.equal (Sanitize.date "2023-12-31") "2023-12-31"
        , test "2023-13-31" <|
            \_ -> Expect.equal (Sanitize.date "2023-13-31") "2023-1"
        , test "2099-12-32" <|
            \_ -> Expect.equal (Sanitize.date "2099-12-32") "2099-12-3"
        ]


parseDate : Test
parseDate =
    describe "Test date parsing"
        [ test "valid date" <|
            \_ ->
                Expect.equal (Sanitize.parseDate "2023-01-23") (Just "2023-01-23")
        , test "invalid year" <|
            \_ ->
                Expect.equal (Sanitize.parseDate "202-01-23") Nothing
        , test "single digit month" <|
            \_ ->
                Expect.equal (Sanitize.parseDate "2023-1-23") Nothing
        , test "single digit day" <|
            \_ ->
                Expect.equal (Sanitize.parseDate "2023-11-8") Nothing
        ]


parseHM : Test
parseHM =
    describe "Test Hour/Minute parsing"
        [ test "single digit min" <|
            \_ ->
                Expect.equal (Sanitize.parseHM "1:2") Nothing
        , test "2-digit hour" <|
            \_ ->
                Expect.equal (Sanitize.parseHM "12:43") (Just "12:43")
        , test "leading 0 min" <|
            \_ ->
                Expect.equal (Sanitize.parseHM "1:02") (Just "1:02")
        , test "leading 0 hour" <|
            \_ ->
                Expect.equal (Sanitize.parseHM "01:02") (Just "01:02")
        , test "min greater 59" <|
            \_ ->
                Expect.equal (Sanitize.parseHM "01:60") Nothing
        , test "hour is 23" <|
            \_ ->
                Expect.equal (Sanitize.parseHM "23:15") (Just "23:15")
        , test "hour is > 23" <|
            \_ ->
                Expect.equal (Sanitize.parseHM "24:23") Nothing
        , test "hour/min is 0" <|
            \_ ->
                Expect.equal (run Sanitize.hmParser "0:00") (Ok "0:00")
        , test "hour/min is 00:00" <|
            \_ ->
                Expect.equal (run Sanitize.hmParser "00:00") (Ok "00:00")
        ]


parseNumber : Test
parseNumber =
    describe "Test number parsing"
        [ test "test simple number" <|
            \_ ->
                Expect.equal (Sanitize.float "1.2") "1.2"
        , test "test leading dec" <|
            \_ ->
                Expect.equal (Sanitize.float ".2") ".2"
        , test "test non digit" <|
            \_ ->
                Expect.equal (Sanitize.float "1.2a") "1.2"
        , test "test negative number" <|
            \_ ->
                Expect.equal (Sanitize.float "-1.2") "-1.2"
        , test "test misplaced dash" <|
            \_ ->
                Expect.equal (Sanitize.float "1-2") "1"
        ]
