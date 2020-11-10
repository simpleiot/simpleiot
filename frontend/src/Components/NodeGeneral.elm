module Components.NodeGeneral exposing (view)

import Api.Node as Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Input as Input
import Time
import UI.Icon as Icon
import UI.Style as Style exposing (colors, size)
import Utils.Duration as Duration
import Utils.Iso8601 as Iso8601


view :
    { isRoot : Bool
    , now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , node : Node
    , onApiDelete : String -> msg
    , onEditNodeDescription : String -> String -> msg
    , onApiPostPoint : String -> Point -> msg
    , onDiscardEditedNodeDescription : msg
    }
    -> Element msg
view o =
    let
        sysState =
            case Point.getPoint o.node.points "" Point.typeSysState 0 of
                Just point ->
                    round point.value

                Nothing ->
                    0

        sysStateIcon =
            case sysState of
                -- not sure why I can't use defines in Node.elm here
                1 ->
                    Icon.power

                2 ->
                    Icon.cloudOff

                3 ->
                    Icon.cloud

                _ ->
                    Element.none

        background =
            case sysState of
                3 ->
                    Style.colors.white

                _ ->
                    Style.colors.gray

        hwVersion =
            case Point.getPoint o.node.points "" Point.typeHwVersion 0 of
                Just point ->
                    "HW: " ++ point.text

                Nothing ->
                    ""

        osVersion =
            case Point.getPoint o.node.points "" Point.typeOSVersion 0 of
                Just point ->
                    "OS: " ++ point.text

                Nothing ->
                    ""

        appVersion =
            case Point.getPoint o.node.points "" Point.typeAppVersion 0 of
                Just point ->
                    "App: " ++ point.text

                Nothing ->
                    ""

        latestPointTime =
            case Point.getLatest o.node.points of
                Just point ->
                    point.time

                Nothing ->
                    Time.millisToPosix 0
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , Background.color background
        , spacing 6
        ]
        [ wrappedRow [ spacing 10 ]
            [ sysStateIcon
            , viewNodeId o.node.id
            , if o.isRoot then
                Icon.x (o.onApiDelete o.node.id)

              else
                Element.none
            , Input.text
                [ Background.color background ]
                { onChange = \d -> o.onEditNodeDescription o.node.id d
                , text = Node.description o.node
                , placeholder = Just <| Input.placeholder [] <| text "node description"
                , label = Input.labelHidden "node description"
                }
            , if o.modified then
                Icon.check
                    (o.onApiPostPoint o.node.id
                        { typ = Point.typeDescription
                        , id = ""
                        , index = 0
                        , time = o.now
                        , value = 0
                        , text = Node.description o.node
                        , min = 0
                        , max = 0
                        }
                    )

              else
                Element.none
            , if o.modified then
                Icon.x o.onDiscardEditedNodeDescription

              else
                Element.none
            ]
        , viewPoints <| Point.filterSpecialPoints o.node.points
        , text ("Last update: " ++ Iso8601.toDateTimeString o.zone latestPointTime)
        , text
            ("Time since last update: "
                ++ Duration.toString
                    (Time.posixToMillis o.now
                        - Time.posixToMillis latestPointTime
                    )
            )
        , if hwVersion /= "" && osVersion /= "" && appVersion /= "" then
            text
                ("Version: "
                    ++ hwVersion
                    ++ " "
                    ++ osVersion
                    ++ " "
                    ++ appVersion
                )

          else
            Element.none
        ]


viewNodeId : String -> Element msg
viewNodeId id =
    el
        [ padding 16
        , size.heading
        ]
    <|
        text id


viewPoints : List Point.Point -> Element msg
viewPoints ios =
    column
        [ padding 16
        , spacing 6
        ]
    <|
        List.map (Point.renderPoint >> text) ios
