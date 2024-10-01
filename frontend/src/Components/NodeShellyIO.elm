module Components.NodeShellyIO exposing (view)

import Api.Point as Point exposing (Point)
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Dict exposing (Dict)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import FormatNumber exposing (format)
import FormatNumber.Locales exposing (Decimals(..), usLocale)
import List
import Round
import Time
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style
import UI.ViewIf exposing (viewIf)
import Utils.Iso8601 as Iso8601


isSettable : List Point -> Bool
isSettable pts =
    List.any (\a -> String.contains "Set" a.typ) pts


view : NodeOptions msg -> Element msg
view o =
    let
        disabled =
            Point.getBool o.node.points Point.typeDisabled ""

        offline =
            Point.getBool o.node.points Point.typeOffline ""

        summaryBackground =
            if disabled || offline then
                Style.colors.ltgray

            else
                Style.colors.none

        typ =
            Point.getText o.node.points Point.typeType ""

        desc =
            Point.getText o.node.points Point.typeDescription ""

        summary =
            "(" ++ typ ++ ")  " ++ desc

        valueElement =
            case
                typ
            of
                "PlusI4" ->
                    i4ValueSummary o.node.points

                _ ->
                    defaultSummary o.node.points
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10, Background.color summaryBackground ]
            [ Icon.io
            , text summary
            , valueElement
            , viewIf disabled <| text "(disabled)"
            , viewIf offline <| text "(offline)"
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        onOffInput =
                            NodeInputs.nodeOnOffInput opts

                        deviceID =
                            Point.getText o.node.points Point.typeDeviceID ""

                        ip =
                            Point.getText o.node.points Point.typeIP ""

                        controlled =
                            Point.getBool o.node.points Point.typeControlled ""

                        latestPointTime =
                            case Point.getLatest o.node.points of
                                Just point ->
                                    point.time

                                Nothing ->
                                    Time.millisToPosix 0
                    in
                    [ textDisplay "ID" deviceID
                    , textLinkDisplay "IP" ip ("http://" ++ ip)
                    , textInput Point.typeDescription "Description" ""
                    , viewIf controlled <| displayControls onOffInput o.node.points
                    , viewIf (isSettable o.node.points) <| checkboxInput Point.typeControlled "Enable Control"
                    , checkboxInput Point.typeDisabled "Disabled"
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    , text ("Last update: " ++ Iso8601.toDateTimeString o.zone latestPointTime)
                    , viewPoints o.zone <| Point.filterSpecialPoints <| List.sortWith Point.sort o.node.points
                    ]

                else
                    []
               )


defaultSummary : List Point -> Element msg
defaultSummary points =
    let
        switches =
            Point.getAll points Point.switch |> List.sortBy .key

        lights =
            Point.getAll points Point.light

        inputs =
            Point.getAll points Point.input
    in
    row []
        [ displayOnOffArray "S:" switches
        , displayOnOffArray "L:" lights
        , displayOnOffArray "I:" inputs
        ]


i4ValueSummary : List Point -> Element msg
i4ValueSummary points =
    let
        valuePoints =
            List.filter (\p -> p.typ == Point.typeValue) points |> List.sortBy .key

        valueElements =
            List.foldl
                (\p ret ->
                    List.append ret [ displayOnOff p ]
                )
                []
                valuePoints
    in
    row [ spacing 8 ] valueElements


displayOnOffArray : String -> List Point -> Element msg
displayOnOffArray label pts =
    if List.length pts > 0 then
        row [] <| text label :: List.map displayOnOff pts

    else
        none


displayOnOff : Point -> Element msg
displayOnOff p =
    let
        v =
            if p.value == 0 then
                "off"

            else
                "on"

        vBackgroundColor =
            if v == "on" then
                Style.colors.blue

            else
                Style.colors.none

        vTextColor =
            if v == "on" then
                Style.colors.white

            else
                Style.colors.black
    in
    el [ paddingXY 7 0, Background.color vBackgroundColor, Font.color vTextColor ] <|
        text <|
            v


displayControls : (String -> String -> String -> String -> Element msg) -> List Point -> Element msg
displayControls onOffInput pts =
    let
        controlTypes =
            [ Point.light, Point.switch ]
    in
    column [] <|
        List.map
            (\t ->
                let
                    tSet =
                        t ++ "Set"

                    ptsFiltered =
                        List.filter (\p -> p.typ == tSet) pts |> List.sortBy .key
                in
                column
                    [ spacing 6
                    ]
                <|
                    List.indexedMap
                        (\i _ ->
                            let
                                key =
                                    String.fromInt i

                                label =
                                    t ++ " " ++ String.fromInt (i + 1)
                            in
                            onOffInput key t tSet label
                        )
                        ptsFiltered
            )
            controlTypes


textDisplay : String -> String -> Element msg
textDisplay label value =
    el [ paddingEach { top = 0, right = 0, bottom = 0, left = 70 } ] <|
        text <|
            label
                ++ ": "
                ++ value


textLinkDisplay : String -> String -> String -> Element msg
textLinkDisplay label value uri =
    el [ paddingEach { top = 0, right = 0, bottom = 0, left = 70 } ] <|
        row []
            [ text <|
                label
                    ++ ": "
            , newTabLink [ Font.underline ] { url = uri, label = text value }
            ]


viewPoints : Time.Zone -> List Point.Point -> Element msg
viewPoints z pts =
    if List.length pts <= 0 then
        Element.none

    else
        let
            formaters =
                metricFormaters z

            fm =
                formatMetric formaters
        in
        table [ padding 7 ]
            { data = List.map fm pts
            , columns =
                let
                    cell =
                        el [ paddingXY 15 5, Border.width 1 ]
                in
                [ { header = cell <| el [ Font.bold, centerX ] <| text "Point"
                  , width = fill
                  , view = \m -> cell <| text m.desc
                  }
                , { header = cell <| el [ Font.bold, centerX ] <| text "Value"
                  , width = fill
                  , view = \m -> cell <| el [ alignRight ] <| text m.value
                  }
                ]
            }


formatMetric : Dict String MetricFormat -> Point.Point -> { desc : String, value : String }
formatMetric formaters p =
    case Dict.get p.typ formaters of
        Just f ->
            { desc = f.desc p, value = f.vf p }

        Nothing ->
            Point.renderPoint2 p


type alias MetricFormat =
    { desc : Point.Point -> String
    , vf : Point.Point -> String
    }


metricFormaters : Time.Zone -> Dict String MetricFormat
metricFormaters _ =
    Dict.fromList
        [ ( "voltage", { desc = descS "Voltage", vf = \p -> Round.round 1 p.value } )
        , ( "temp", { desc = descS "Temperature (C)", vf = \p -> Round.round 1 p.value } )
        , ( "power", { desc = descS "Power", vf = \p -> Round.round 2 p.value } )
        , ( "current", { desc = descS "Current", vf = \p -> Round.round 2 p.value } )
        , ( "brightness", { desc = descS "Brightness", vf = toWhole } )
        , ( "lightTemp", { desc = descS "Light Temperature", vf = toWhole } )
        , ( "transition", { desc = descS "Transition", vf = toWhole } )
        , ( "white", { desc = descS "White", vf = toWhole } )
        ]


descS : String -> Point.Point -> String
descS d _ =
    d


toWhole : Point.Point -> String
toWhole p =
    format { usLocale | decimals = Exact 0 } p.value
