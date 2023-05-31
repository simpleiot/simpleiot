module UI.NodeInputs exposing
    ( NodeInputOptions
    , nodeCheckboxInput
    , nodeCounterWithReset
    , nodeNumberInput
    , nodeOnOffInput
    , nodeOptionInput
    , nodePasteButton
    , nodeTextInput
    , nodeTimeDateInput
    )

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Color
import Element exposing (..)
import Element.Input as Input
import List.Extra
import Round
import Svg as S
import Svg.Attributes as Sa
import Time
import Time.Extra
import UI.Button
import UI.Form as Form
import UI.Sanitize as Sanitize
import UI.Style as Style
import Utils.Time exposing (scheduleToLocal, scheduleToUTC)


type alias NodeInputOptions msg =
    { onEditNodePoint : List Point -> msg
    , node : Node
    , now : Time.Posix
    , zone : Time.Zone
    , labelWidth : Int
    }


nodeTextInput :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> String
    -> Element msg
nodeTextInput o key typ lbl placeholder =
    Input.text
        []
        { onChange =
            \d ->
                o.onEditNodePoint [ Point typ key o.now 0 0 d 0 ]
        , text = Point.getText o.node.points typ key
        , placeholder = Just <| Input.placeholder [] <| text placeholder
        , label = Input.labelLeft [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeTimeDateInput : NodeInputOptions msg -> Int -> Element msg
nodeTimeDateInput o labelWidth =
    let
        zoneOffset =
            Time.Extra.toOffset o.zone o.now

        sModel =
            pointsToSchedule o.node.points

        sDisp =
            checkScheduleToLocal zoneOffset sModel

        send updateSchedule d =
            let
                dClean =
                    Sanitize.time d
            in
            updateSchedule sDisp dClean
                |> checkScheduleToUTC zoneOffset
                |> scheduleToPoints o.now
                |> o.onEditNodePoint

        weekdayCheckboxInput index label =
            Input.checkbox []
                { onChange =
                    \d ->
                        updateScheduleWkday sDisp index d
                            |> checkScheduleToUTC zoneOffset
                            |> scheduleToPoints o.now
                            |> o.onEditNodePoint
                , checked = List.member index sDisp.weekdays
                , icon = Input.defaultCheckbox
                , label = Input.labelAbove [] <| text label
                }
    in
    column [ spacing 5 ]
        [ wrappedRow
            [ spacing 20
            , paddingEach { top = 0, right = 0, bottom = 5, left = 0 }
            ]
            -- here, number matches Go Weekday definitions
            -- https://pkg.go.dev/time#Weekday
            [ el [ width <| px (o.labelWidth - 120) ] none
            , text "Weekdays:"
            , weekdayCheckboxInput 0 "S"
            , weekdayCheckboxInput 1 "M"
            , weekdayCheckboxInput 2 "T"
            , weekdayCheckboxInput 3 "W"
            , weekdayCheckboxInput 4 "T"
            , weekdayCheckboxInput 5 "F"
            , weekdayCheckboxInput 6 "S"
            ]
        , Input.text
            []
            { label = Input.labelLeft [ width (px labelWidth) ] <| el [ alignRight ] <| text <| "Start time:"
            , onChange = send (\sched d -> { sched | startTime = d })
            , text = sDisp.startTime
            , placeholder = Nothing
            }
        , Input.text
            []
            { label = Input.labelLeft [ width (px labelWidth) ] <| el [ alignRight ] <| text <| "End time:"
            , onChange = send (\sched d -> { sched | endTime = d })
            , text = sDisp.endTime
            , placeholder = Nothing
            }
        , el [ Element.paddingEach { top = 0, bottom = 0, right = 0, left = labelWidth - 59 } ] <| text "Dates:"
        , el [ Element.paddingEach { top = 0, bottom = 0, right = 0, left = labelWidth - 59 } ] <|
            Form.button
                { label = "Add Date"
                , color = Style.colors.blue
                , onPress =
                    o.onEditNodePoint
                        [ { typ = Point.typeDate
                          , index = 0
                          , key = ""
                          , text = ""
                          , time = o.now
                          , tombstone = 0
                          , value = 0
                          }
                        ]
                }
        ]


pointsToSchedule : List Point -> Utils.Time.Schedule
pointsToSchedule points =
    let
        start =
            Point.getText points Point.typeStart ""

        end =
            Point.getText points Point.typeEnd ""

        weekdays =
            List.filter
                (\d ->
                    let
                        dString =
                            String.fromInt d

                        p =
                            Point.getValue points Point.typeWeekday dString
                    in
                    p == 1
                )
                [ 0, 1, 2, 3, 4, 5, 6 ]
    in
    { startTime = start
    , endTime = end
    , weekdays = weekdays
    }


scheduleToPoints : Time.Posix -> Utils.Time.Schedule -> List Point
scheduleToPoints now sched =
    [ Point Point.typeStart "" now 0 0 sched.startTime 0
    , Point Point.typeEnd "" now 0 0 sched.endTime 0
    ]
        ++ List.map
            (\wday ->
                if List.member wday sched.weekdays then
                    Point Point.typeWeekday (String.fromInt wday) now (toFloat wday) 1 "" 0

                else
                    Point Point.typeWeekday (String.fromInt wday) now (toFloat wday) 0 "" 0
            )
            [ 0, 1, 2, 3, 4, 5, 6 ]



-- only convert to utc if both times are valid


checkScheduleToUTC : Int -> Utils.Time.Schedule -> Utils.Time.Schedule
checkScheduleToUTC offset sched =
    if validHM sched.startTime && validHM sched.endTime then
        scheduleToUTC offset sched

    else
        sched


updateScheduleWkday : Utils.Time.Schedule -> Int -> Bool -> Utils.Time.Schedule
updateScheduleWkday sched index checked =
    let
        weekdays =
            if checked then
                if List.member index sched.weekdays then
                    sched.weekdays

                else
                    index :: sched.weekdays

            else
                List.Extra.remove index sched.weekdays
    in
    { sched | weekdays = List.sort weekdays }



-- only convert to local if both times are valid


checkScheduleToLocal : Int -> Utils.Time.Schedule -> Utils.Time.Schedule
checkScheduleToLocal offset sched =
    if validHM sched.startTime && validHM sched.endTime then
        scheduleToLocal offset sched

    else
        sched


validHM : String -> Bool
validHM t =
    case Sanitize.parseHM t of
        Just _ ->
            True

        Nothing ->
            False


nodeCheckboxInput :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> Element msg
nodeCheckboxInput o key typ lbl =
    Input.checkbox
        []
        { onChange =
            \d ->
                let
                    v =
                        if d then
                            1.0

                        else
                            0.0
                in
                o.onEditNodePoint
                    [ Point typ key o.now 0 v "" 0 ]
        , checked =
            Point.getValue o.node.points typ key == 1
        , icon = Input.defaultCheckbox
        , label =
            if lbl /= "" then
                Input.labelLeft [ width (px o.labelWidth) ] <|
                    el [ alignRight ] <|
                        text <|
                            lbl
                                ++ ":"

            else
                Input.labelHidden ""
        }


nodeNumberInput :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> Element msg
nodeNumberInput o key typ lbl =
    let
        pMaybe =
            Point.get o.node.points typ key

        currentValue =
            case pMaybe of
                Just p ->
                    if p.text /= "" then
                        if p.text == Point.blankMajicValue || p.text == "blank" then
                            ""

                        else
                            Sanitize.float p.text

                    else
                        String.fromFloat (Round.roundNum 6 p.value)

                Nothing ->
                    ""

        currentValueF =
            case pMaybe of
                Just p ->
                    p.value

                Nothing ->
                    0
    in
    Input.text
        []
        { onChange =
            \d ->
                let
                    dCheck =
                        if d == "" then
                            Point.blankMajicValue

                        else
                            case String.toFloat d of
                                Just _ ->
                                    d

                                Nothing ->
                                    currentValue

                    v =
                        if dCheck == Point.blankMajicValue then
                            0

                        else
                            Maybe.withDefault currentValueF <| String.toFloat dCheck
                in
                o.onEditNodePoint
                    [ Point typ key o.now 0 v dCheck 0 ]
        , text = currentValue
        , placeholder = Nothing
        , label = Input.labelLeft [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeOptionInput :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> List ( String, String )
    -> Element msg
nodeOptionInput o key typ lbl options =
    Input.radio
        [ spacing 6 ]
        { onChange =
            \sel ->
                o.onEditNodePoint
                    [ Point typ key o.now 0 0 sel 0 ]
        , label =
            Input.labelLeft [ padding 12, width (px o.labelWidth) ] <|
                el [ alignRight ] <|
                    text <|
                        lbl
                            ++ ":"
        , selected = Just <| Point.getText o.node.points typ key
        , options =
            List.map
                (\opt ->
                    Input.option (Tuple.first opt) (text (Tuple.second opt))
                )
                options
        }


nodeCounterWithReset :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> String
    -> Element msg
nodeCounterWithReset o key typ pointResetName lbl =
    let
        currentValue =
            Point.getValue o.node.points typ key

        currentResetValue =
            Point.getValue o.node.points pointResetName key /= 0
    in
    row [ spacing 20 ]
        [ el [ width (px o.labelWidth) ] <|
            el [ alignRight ] <|
                text <|
                    lbl
                        ++ ": "
                        ++ String.fromFloat currentValue
        , Input.checkbox []
            { onChange =
                \v ->
                    let
                        vFloat =
                            if v then
                                1.0

                            else
                                0
                    in
                    o.onEditNodePoint [ Point pointResetName key o.now 0 vFloat "" 0 ]
            , icon = Input.defaultCheckbox
            , checked = currentResetValue
            , label =
                Input.labelLeft [] (text "reset")
            }
        ]


nodeOnOffInput :
    NodeInputOptions msg
    -> String
    -> String
    -> String
    -> String
    -> Element msg
nodeOnOffInput o key typ pointSetName lbl =
    let
        currentValue =
            Point.getValue o.node.points typ key

        currentSetValue =
            Point.getValue o.node.points pointSetName key

        fill =
            if currentSetValue == 0 then
                Color.rgb 0.5 0.5 0.5

            else
                Color.rgb255 50 100 150

        fillS =
            Color.toCssString fill

        offset =
            if currentSetValue == 0 then
                3

            else
                3 + 24

        newValue =
            if currentSetValue == 0 then
                1

            else
                0
    in
    row [ spacing 10 ]
        [ el [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        , Input.button
            []
            { onPress = Just <| o.onEditNodePoint [ Point pointSetName key o.now 0 newValue "" 0 ]
            , label =
                el [ width (px 100) ] <|
                    html <|
                        S.svg [ Sa.viewBox "0 0 48 24" ]
                            [ S.rect
                                [ Sa.x "0"
                                , Sa.y "0"
                                , Sa.width "48"
                                , Sa.height "24"
                                , Sa.ry "3"
                                , Sa.rx "3"
                                , Sa.fill fillS
                                ]
                              <|
                                if currentValue /= currentSetValue then
                                    let
                                        fillFade =
                                            if currentSetValue == 0 then
                                                Color.rgb 0.9 0.9 0.9

                                            else
                                                Color.rgb255 150 200 255

                                        fillFadeS =
                                            Color.toCssString fillFade
                                    in
                                    [ S.animate
                                        [ Sa.attributeName "fill"
                                        , Sa.dur "2s"
                                        , Sa.repeatCount "indefinite"
                                        , Sa.values <|
                                            fillFadeS
                                                ++ ";"
                                                ++ fillS
                                                ++ ";"
                                                ++ fillFadeS
                                        ]
                                        []
                                    ]

                                else
                                    []
                            , S.rect
                                [ Sa.x (String.fromFloat offset)
                                , Sa.y "3"
                                , Sa.width "18"
                                , Sa.height "18"
                                , Sa.ry "3"
                                , Sa.rx "3"
                                , Sa.fill (Color.toCssString Color.white)
                                ]
                                []
                            ]
            }
        ]


nodePasteButton :
    NodeInputOptions msg
    -> Element msg
    -> String
    -> String
    -> Element msg
nodePasteButton o label typ value =
    row [ spacing 10, paddingEach { top = 0, bottom = 0, right = 0, left = 75 } ]
        [ UI.Button.clipboard <| o.onEditNodePoint [ Point typ "" o.now 0 0 value 0 ]
        , label
        ]
