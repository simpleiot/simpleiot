module Sanitize exposing (date, parseHM)

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
